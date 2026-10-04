package data

import (
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"

	"github.com/braginantonev/mhserver/internal/repository"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/pkg/contextkeys"
	pb "github.com/braginantonev/mhserver/proto/gen/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	services.Service

	cfg         DataServiceConfig
	activeFiles *CachedFiles
	sem         repository.Semaphore
}

func NewDataServer(ctx context.Context, cfg DataServiceConfig) *DataServer {
	return &DataServer{
		cfg:         cfg,
		activeFiles: NewCachedFiles(ctx),
		sem:         repository.NewSemaphore(SEMAPHORE_SIZE),
	}
}

func (s *DataServer) InitFile(ctx context.Context, req_file *pb.RequiredFile) (*pb.InitInfo, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	filepath, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, req_file.Dir.Value, s.cfg.ServiceName)
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

	max_chunk_size := s.cfg.Memory.MaxChunkSize // cfg chunk size must be rounded to RAM page
	if file_size <= s.cfg.Memory.MinChunkSize {
		max_chunk_size = file_size
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

func (s *DataServer) SaveFile(stream pb.DataService_SaveFileServer) error {
	defer s.sem.Release()
	s.sem.Acquire()

	// init file id
	init_info, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		slog.ErrorContext(stream.Context(), "failed recv stream", slog.Any("error", err))
		return ErrInternal
	}

	meta := init_info.GetMeta()
	if meta == nil {
		return ErrBrokenSequence
	}

	username, ok := stream.Context().Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(stream.Context(), "failed get username from context", slog.Any("got", stream.Context().Value(contextkeys.USERNAME)))
		return ErrInternal
	}

	if !DirIsCorrect(meta.Dir.GetValue()) {
		return ErrBadDirSyntax
	}

	var file *os.File
	var file_size uint64
	if meta.NewSize == nil { // update file, no create. We search file in uspace's
		for _, usp := range s.cfg.UserSpaces {
			file, err = os.OpenFile(filepath.Join(CompileServiceDir(s.cfg.WorkspacePath, usp, username, meta.Dir.Value, SERVICE_NAME), meta.Name), os.O_WRONLY, 0660)
			if err == nil {
				break
			}

			if os.IsNotExist(err) {
				continue
			}

			slog.ErrorContext(stream.Context(), "failed open file to write only", slog.Any("error", err))
			return ErrInternal
		}

		if file == nil {
			return ErrFileNotExist
		}

		stat, err := file.Stat()
		if err != nil {
			slog.ErrorContext(stream.Context(), "failed get stat of exist file", slog.Any("error", err))
			return ErrInternal
		}

		file_size = uint64(stat.Size())

	} else { // create and truncate file. We find uspace, where available space must be enough for file
		var available_uspace string
		for _, usp := range s.cfg.UserSpaces {
			space, err := freemem.GetAvailableDiskSpace(filepath.Join(s.cfg.WorkspacePath, usp))
			if err != nil {
				slog.ErrorContext(stream.Context(), "failed ge t available disk space in uspace", slog.Any("error", err))
				return ErrInternal
			}

			if space >= *meta.NewSize {
				available_uspace = usp
				break
			}
		}

		if len(available_uspace) == 0 {
			return ErrNotEnoughDiskSpace
		}

		if !FileIsCorrect(meta.Name) {
			return ErrBadFilenameSyntax
		}

		file, err = os.OpenFile(filepath.Join(CompileServiceDir(s.cfg.WorkspacePath, available_uspace, username, meta.Dir.Value, SERVICE_NAME), meta.Name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
		if err != nil {
			slog.ErrorContext(stream.Context(), "failed create file", slog.Any("error", err))
			return ErrInternal
		}

		if err = file.Truncate(int64(*meta.NewSize)); err != nil {
			slog.ErrorContext(stream.Context(), "failed truncate file", slog.Any("error", err))
			return ErrInternal
		}

		file_size = *meta.NewSize
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			if err = file.Sync(); err != nil {
				slog.ErrorContext(stream.Context(), "failed sync file", slog.Any("error", err))
				return ErrInternal
			}
			return stream.SendAndClose(&emptypb.Empty{})
		}

		chunk := req.GetChunk()
		if chunk == nil {
			return ErrBrokenSequence
		}

		if err != nil {
			return err
		}

		if chunk.Offset+uint64(len(chunk.Data)) > file_size {
			return ErrUnexpectedFileChange
		}

		if _, err = file.WriteAt(chunk.Data, int64(chunk.Offset)); err != nil {
			slog.ErrorContext(stream.Context(), "failed write chunk to file", slog.Any("error", err))
			return ErrInternal
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

	chunks_count := uint64(math.Ceil(float64(file.Meta.Size) / float64(s.cfg.Memory.MaxChunkSize)))
	data := make([]byte, 0, s.cfg.Memory.MaxChunkSize)

readLoop:
	for i := range chunks_count {
		select {
		case <-stream.Context().Done():
			break readLoop
		default:
			n, err := file.Read(data)
			if err != nil {
				slog.ErrorContext(stream.Context(), "failed read file", slog.Any("error", err))
				return ErrInternal
			}

			stream.Send(&pb.Chunk{
				Offset: i * s.cfg.Memory.MaxChunkSize,
				Data:   data[:n],
			})
		}
	}

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
	if _, err := io.Copy(hash, file); err != nil {
		slog.ErrorContext(ctx, "failed copy file for sum calc", slog.Any("error", err))
		return nil, ErrInternal
	}

	return &pb.SHASum{Value: hash.Sum(nil)[:]}, nil
}

func (s *DataServer) GetAvailableDiskSpace(ctx context.Context, dir *pb.Directory) (*pb.Size, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, "/", s.cfg.ServiceName)
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

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, dir.Value, s.cfg.ServiceName)
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

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, dir.Value, s.cfg.ServiceName)
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

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	dir_path, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, dir.Value, s.cfg.ServiceName)
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

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	filepath, err := dirs.GetDataPath(s.cfg.WorkspacePath, username, req_file.Dir.Value, s.cfg.ServiceName)
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
