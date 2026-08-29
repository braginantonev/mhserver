package data

import (
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"math"
	"os"
	"sync"

	"github.com/braginantonev/mhserver/internal/repository"
	"github.com/braginantonev/mhserver/internal/repository/dirs"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	pb "github.com/braginantonev/mhserver/proto/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	cfg         DataServiceConfig
	activeFiles *CachedFiles
	sem         repository.Semaphore
}

func NewDataServer(ctx context.Context, cfg DataServiceConfig) *DataServer {
	sem_size := (cfg.Memory.Allocated * 985 / 1000) / cfg.Memory.MaxChunkSize
	slog.Info("Set semaphore size", slog.String("subserver", string(cfg.ServiceName)), slog.Int("value", int(sem_size)))

	return &DataServer{
		cfg:         cfg,
		activeFiles: NewCachedFiles(ctx),
		sem:         repository.NewSemaphore(int(sem_size)),
	}
}

func (s *DataServer) getChunk(ctx context.Context, reader io.ReaderAt, offset int64) ([]byte, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	data := make([]byte, offset)
	n, err := reader.ReadAt(data, int64(offset))
	if err != nil && err != io.EOF {
		slog.ErrorContext(ctx, "failed read file chunk", slog.Any("err", err))
		return nil, ErrInternal
	}

	if n == 0 && err == io.EOF {
		return nil, ErrReadOutOfFile
	}

	return data[:n], nil
}

func (s *DataServer) InitFile(ctx context.Context, req_file *pb.RequiredFile) (*pb.InitInfo, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	filepath, err := dirs.GetDataPath(s.cfg.WorkspacePath, req_file.Dir.User, req_file.Dir.Value, s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	if !dirs.FileIsCorrect(req_file.Name) {
		return nil, ErrBadFilenameSyntax
	}

	file, err := os.OpenFile(filepath+req_file.Name, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDirNotFound
		}

		slog.ErrorContext(ctx, "failed open file to read", slog.Any("err", err))
		return nil, ErrInternal
	}

	var file_size uint64

	if req_file.NewSize != nil {
		file_size = *req_file.NewSize
		if err := file.Truncate(int64(*req_file.NewSize)); err != nil {
			slog.ErrorContext(ctx, "failed truncate file size", slog.Any("err", err))
			return nil, ErrInternal
		}
	} else {
		file_stat, err := file.Stat()
		if err != nil {
			slog.ErrorContext(ctx, "failed get file stat", slog.Any("err", err))
			return nil, ErrInternal
		}
		file_size = uint64(file_stat.Size())
	}

	var max_chunk_size uint64
	if file_size <= s.cfg.Memory.MinChunkSize {
		max_chunk_size = file_size
	} else {
		file_based := uint64(float64(BASE_CHUNK_SIZE) * math.Log2(float64(file_size)/float64(BASE_CHUNK_SIZE)+1))
		max_chunk_size = max(s.cfg.Memory.MinChunkSize, min(file_based, s.cfg.Memory.MaxChunkSize))
	}

	// Round to RAM page
	if max_chunk_size > 4096 {
		max_chunk_size = (max_chunk_size / 4096) * 4096
	}

	return &pb.InitInfo{
		FileID: &pb.FileID{
			Value: s.activeFiles.Push(NewFile(file, FileMeta{
				Size: file_size,
			})).String(),
		},
		MaxChunkSize: max_chunk_size,
	}, nil
}

func (s *DataServer) GetFileChunk(ctx context.Context, chunk *pb.GetChunk) (*pb.Chunk, error) {
	// used semaphore from s.getChunk() method later

	uuid, err := uuid.Parse(chunk.Id.Value)
	if err != nil {
		return nil, ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	data, err := s.getChunk(ctx, file, int64(chunk.Offset))

	return &pb.Chunk{
		Data:   data,
		Offset: chunk.Offset,
	}, err
}

func (s *DataServer) SaveFile(stream pb.DataService_SaveFileServer) error {
	defer s.sem.Release()
	s.sem.Acquire()

	// init file id
	first_chunk, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		slog.ErrorContext(stream.Context(), "failed recv stream", slog.Any("error", err))
		return ErrInternal
	}

	id, err := uuid.Parse(first_chunk.Id.Value)
	if err != nil {
		return ErrBadUUID
	}

	file, ok := s.activeFiles.Get(id)
	if !ok {
		return ErrConnectionNotFound
	}
	defer file.Close()

	save_pool := NewSaveWorkersPool(stream.Context(), s.sem, file)
	if err = save_pool.TryPush(first_chunk.GetValue()); err != nil { // save first chunk
		return err
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			if err = save_pool.Flush(); err != nil {
				return err
			}
			if err = save_pool.Recover(); err != nil { // end error check
				return err
			}
			return stream.SendAndClose(&emptypb.Empty{})
		}

		if err != nil {
			return err
		}

		if req.Id.GetValue() != first_chunk.Id.Value {
			return ErrUnexpectedFileChange
		}

		if err = save_pool.TryPush(req.GetValue()); err != nil {
			return err
		}
	}
}

func (s *DataServer) ReadFile(id *pb.FileID, stream pb.DataService_ReadFileServer) error {
	defer s.sem.Release()
	s.sem.Acquire()

	uuid, err := uuid.Parse(id.Value)
	if err != nil {
		return ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return ErrConnectionNotFound
	}

	errors := make(chan error, 1)
	wg := sync.WaitGroup{}

	read := func(offset uint64) {
		defer func() {
			s.sem.Release()
			wg.Done()
		}()
		s.sem.Acquire()

		data, err := s.getChunk(stream.Context(), file, int64(offset))
		if err != nil {
			if err := stream.SendMsg(err); err != nil {
				errors <- err
			}
			return
		}

		if err := stream.Send(&pb.Chunk{
			Data:   data,
			Offset: offset,
		}); err != nil {
			errors <- err
		}
	}

	for i := uint64(0); i < uint64(math.Ceil(float64(file.Meta.Size)/float64(s.cfg.Memory.MaxChunkSize))); i++ {
		select {
		case err := <-errors:
			return err
		default:
			wg.Add(1)
			go read(i * s.cfg.Memory.MaxChunkSize)
		}
	}

	wg.Wait()
	return nil
}

func (s *DataServer) GetSum(ctx context.Context, id *pb.FileID) (*pb.SHASum, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	uuid, err := uuid.Parse(id.Value)
	if err != nil {
		return nil, ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	hash := sha256.New()
	for i := uint64(0); i < uint64(math.Ceil(float64(file.Meta.Size)/float64(s.cfg.Memory.MaxChunkSize))); i++ {
		body := make([]byte, s.cfg.Memory.MaxChunkSize)
		n, err := file.ReadAt(body, int64(s.cfg.Memory.MaxChunkSize)*int64(i))
		if err != nil && err != io.EOF {
			slog.ErrorContext(ctx, "failed read file chunk", slog.Any("err", err))
			return nil, ErrInternal
		}
		_, _ = hash.Write(body[:n])
	}

	return &pb.SHASum{Value: hash.Sum(nil)[:]}, nil
}

func (s *DataServer) GetAvailableDiskSpace(ctx context.Context, dir *pb.Directory) (*pb.Size, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, dir.User, "/", s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	space, err := freemem.GetAvailableDiskSpace(dir_path)
	if err != nil {
		return nil, ErrDirNotFound
	}

	return &pb.Size{Value: space}, nil
}

func (s *DataServer) GetFiles(ctx context.Context, dir *pb.Directory) (*pb.FilesList, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, dir.User, dir.Value, s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir_path)
	if err != nil {
		return nil, ErrDirNotFound
	}

	list := &pb.FilesList{
		Value: make([]*pb.FileInfo, len(files)),
	}

	for i, file := range files {
		list.Value[i] = &pb.FileInfo{
			Name:  file.Name(),
			IsDir: file.IsDir(),
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		list.Value[i].Size = uint64(info.Size())
		list.Value[i].ModTime = uint64(info.ModTime().Unix())
	}

	return list, nil
}

func (s *DataServer) CreateDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, dir.User, dir.Value, s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir_path, 0700); err != nil {
		if os.IsExist(err) {
			return nil, ErrDirAlreadyExist
		}

		slog.ErrorContext(ctx, "failed create user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return nil, nil
}

func (s *DataServer) RemoveDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, dir.User, dir.Value, s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(dir_path); err != nil {
		slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return nil, nil
}

func (s *DataServer) RemoveFile(ctx context.Context, req_file *pb.RequiredFile) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	filepath, err := dirs.GetDataPath(s.cfg.WorkspacePath, req_file.Dir.User, req_file.Dir.Value, s.cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	if !dirs.FileIsCorrect(req_file.Name) {
		return nil, ErrBadFilenameSyntax
	}

	filepath += req_file.Name

	if err := os.Remove(filepath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotExist
		}

		slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return nil, nil
}
