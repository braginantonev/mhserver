package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/repository/fileuuidmap"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	pb "github.com/braginantonev/mhserver/proto/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	cfg         dataconfig.DataServiceConfig
	activeFiles *fileuuidmap.FileUUIDMap
	sem         chan any
}

func NewDataServer(ctx context.Context, cfg dataconfig.DataServiceConfig) *DataServer {
	sem_size := int(cfg.Memory.AvailableRAM / cfg.Memory.MaxChunkSize)

	slog.Info("Set semaphore size", "value", sem_size)

	return &DataServer{
		cfg:         cfg,
		activeFiles: fileuuidmap.NewFileUUIDMap(ctx),
		sem:         make(chan any, sem_size),
	}
}

func (s *DataServer) getDataPath(user, dir string, data_type pb.FileType) (string, error) {
	filetype, ok := catalogs[data_type]
	if !ok {
		return "", ErrUnexpectedFileType
	}

	if dir == "" {
		return "", ErrUnspecifiedDir
	}

	if dir[0] != '/' || strings.Contains(dir, "..") {
		return "", ErrBadDirSyntax
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	return fmt.Sprintf("%s%s/%s%s", s.cfg.WorkspacePath, user, filetype, dir), nil
}

func (s *DataServer) CreateConnection(ctx context.Context, req *pb.ConnectionRequest) (*pb.Connection, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	file_path, err := s.getDataPath(req.Username, req.Directory, req.Filetype)
	if err != nil {
		return nil, err
	}

	file_path += req.Filename

	r_stat, err := os.Stat(file_path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.ErrorContext(ctx, "failed get stat from file", slog.Any("err", err))
		return nil, ErrInternal
	}

	file_size := req.Size

	if r_stat != nil {
		// File exist, but user want update him
		if req.Size != 0 {
			disk_space, err := freemem.GetAvailableDiskSpace(s.cfg.WorkspacePath)
			if err != nil {
				slog.ErrorContext(ctx, "failed get available disk space", slog.Any("err", err))
				return nil, ErrInternal
			}

			if disk_space-s.activeFiles.ExpectedSavedSpace() < req.Size {
				return nil, ErrNotEnoughDiskSpace
			}

			err = os.Remove(file_path)
			if err != nil {
				slog.ErrorContext(ctx, "failed remove existed file on save connection", slog.Any("err", err))
				return nil, ErrInternal
			}
		} else {
			file_size = uint64(r_stat.Size())
		}
	}

	// Open file to read. If file not exist -> we use info in request
	file, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrDirNotFound
		}

		slog.ErrorContext(ctx, "failed open file to read", slog.Any("err", err))
		return nil, ErrInternal
	}

	var chunk_size uint64
	if file_size < s.cfg.Memory.MinChunkSize {
		chunk_size = file_size
	} else {
		file_based := uint64(float64(dataconfig.BASE_CHUNK_SIZE) * math.Log2(float64(file_size)/float64(dataconfig.BASE_CHUNK_SIZE)+1))
		chunk_size = max(s.cfg.Memory.MinChunkSize, min(file_based, s.cfg.Memory.MaxChunkSize))
	}

	// Round to RAM page
	if chunk_size > 4096 {
		chunk_size = (chunk_size / 4096) * 4096
	}

	chunks_count := int(math.Ceil(float64(file_size) / float64(chunk_size)))

	uuid := s.activeFiles.Push(fileuuidmap.NewFile(file, file_path, fileuuidmap.NewChunksInfo(chunk_size, chunks_count)))

	return &pb.Connection{
		UUID:        uuid.String(),
		ChunkSize:   chunk_size,
		ChunksCount: int32(chunks_count),
	}, nil
}

func (s *DataServer) GetData(ctx context.Context, chunk *pb.GetChunk) (*pb.FilePart, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	uuid, err := uuid.Parse(chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	offset := int64(file.GetChunkSize()) * int64(chunk.ChunkId)

	read_data := make([]byte, file.GetChunkSize())
	n, err := file.ReadAt(read_data, offset)
	if err != nil && err != io.EOF {
		slog.ErrorContext(ctx, "failed read file chunk", slog.Any("err", err))
		return nil, ErrInternal
	}

	if n == 0 && err == io.EOF {
		return nil, ErrReadOutOfFile
	}

	return &pb.FilePart{
		Chunk:  read_data[:n],
		Offset: offset,
	}, nil
}

func (s *DataServer) SaveData(ctx context.Context, chunk *pb.SaveChunk) (*emptypb.Empty, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	uuid, err := uuid.Parse(chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	if file.IsLoaded() {
		return nil, ErrUnexpectedFileChange
	}

	if len(chunk.Data.Chunk) > int(file.GetChunkSize()) {
		return nil, ErrIncorrectChunkSize
	}

	_, err = file.WriteAt(chunk.Data.Chunk, chunk.Data.Offset)
	if err != nil {
		slog.ErrorContext(ctx, "failed write chunk to file", slog.Any("err", err))
		return nil, ErrInternal
	}

	_ = s.activeFiles.UpdateLoadedChunks(uuid)

	return nil, nil
}

func (s *DataServer) GetSum(ctx context.Context, chunk *pb.GetChunk) (*pb.SHASum, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	uuid, err := uuid.Parse(chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrConnectionNotFound
	}

	body := make([]byte, file.GetChunkSize())
	n, err := file.ReadAt(body, int64(file.GetChunkSize())*int64(chunk.ChunkId))
	if err != nil && err != io.EOF {
		slog.ErrorContext(ctx, "failed read file chunk", slog.Any("err", err))
		return nil, ErrInternal
	}

	if n == 0 && err == io.EOF {
		return nil, ErrReadOutOfFile
	}

	sha := sha256.Sum256(body[:n])
	return &pb.SHASum{Value: sha[:]}, nil
}

func (s *DataServer) GetAvailableDiskSpace(ctx context.Context, dir *pb.Directory) (*pb.Size, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	dir_path, err := s.getDataPath(dir.User, "/", pb.FileType_File)
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
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	dir_path, err := s.getDataPath(dir.User, dir.Value, pb.FileType_File)
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
		list.Value[i].ModTime = info.ModTime().Unix()
	}

	return list, nil
}

func (s *DataServer) CreateDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	dir_path, err := s.getDataPath(dir.User, dir.Value, pb.FileType_File)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir_path, 0600); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrDirAlreadyExist
		}

		slog.ErrorContext(ctx, "failed create user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return nil, nil
}

func (s *DataServer) RemoveDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	dir_path, err := s.getDataPath(dir.User, dir.Value, pb.FileType_File)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(dir_path); err != nil {
		slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return nil, nil
}
