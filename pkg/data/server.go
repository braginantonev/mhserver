package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	STANDARD_CHUNK_SIZE int = 50
)

type Config struct {
	// User files path
	WorkspacePath string

	// Chunk size is a size of one part file, which will be saved
	ChunkSize int

	//Todo: Upload semaphore
}

func NewDataServerConfig(workspace_path string, chunk_size int) Config {
	if chunk_size <= 0 {
		chunk_size = STANDARD_CHUNK_SIZE
	}

	return Config{
		WorkspacePath: workspace_path,
		ChunkSize:     chunk_size,
	}
}

type DataServer struct {
	pb.DataServiceServer
	cfg   Config
	cache *Cache
}

func NewDataServer(ctx context.Context, cfg Config) *DataServer {
	return &DataServer{
		cfg:   cfg,
		cache: NewCache(ctx),
	}
}

func (s *DataServer) GetData(ctx context.Context, data *pb.Data) (*pb.FilePart, error) {
	if data.Action != pb.Action_Get {
		return nil, ErrWrongAction
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s", s.cfg.WorkspacePath, data.Info.User, dataFolders[data.Info.Type], data.Info.File)

	file, ok := s.cache.Get(file_path)
	if !ok {
		var err error
		file, err = os.OpenFile(file_path, os.O_RDONLY, 0220)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, ErrFileNotExist
			}
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		s.cache.Push(file_path, file)
	}

	send_data := make([]byte, s.cfg.ChunkSize)
	n, err := file.ReadAt(send_data, data.Part.Offset)
	if err != nil {
		if err == io.EOF {
			return nil, EOF
		}
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return &pb.FilePart{
		Body:   send_data[:n],
		Offset: data.Part.Offset,
	}, nil
}

func (s *DataServer) SaveData(ctx context.Context, data *pb.Data) (*emptypb.Empty, error) {
	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s.part", s.cfg.WorkspacePath, data.Info.User, dataFolders[data.Info.Type], data.Info.File)

	switch data.Action {
	case pb.Action_Create:
		file, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0660)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		s.cache.Push(file_path, file)
		slog.InfoContext(ctx, "Create file - "+file_path)

	case pb.Action_Patch:
		var err error
		file, ok := s.cache.Get(file_path)
		if !ok {
			file, err = os.OpenFile(file_path, os.O_WRONLY, 0440)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil, ErrFileNotExist
				}
				return nil, fmt.Errorf("%w: %v", ErrInternal, err)
			}
			s.cache.Push(file_path, file)
		}

		_, err = file.WriteAt(data.Part.Body, data.Part.Offset)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}

	case pb.Action_Finish:
		err := os.Rename(file_path, file_path[:len(file_path)-5]) // file_path[:len(file_path)-5] -> del ".part"
		if err != nil {
			return nil, ErrFileNotExist
		}
		slog.InfoContext(ctx, "Rename - "+file_path)

	default:
		return nil, ErrWrongAction
	}

	return nil, nil
}
