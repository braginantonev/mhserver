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

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/repository/filecache"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	cfg          dataconfig.DataServiceConfig
	cache        *filecache.Cache
	active_files map[string]any
}

func NewDataServer(ctx context.Context, cfg dataconfig.DataServiceConfig) *DataServer {
	return &DataServer{
		cfg:   cfg,
		cache: filecache.NewCache(ctx),
	}
}

func (s *DataServer) openFile(path string, flag int, perm os.FileMode) (file *os.File, err error) {
	file, ok := s.cache.Get(path)
	if !ok {
		file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, ErrFileNotExist
			}
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		s.cache.Push(path, file)
	}
	return file, err
}

func (s *DataServer) GetData(ctx context.Context, data *pb.Data) (*pb.FilePart, error) {
	if data.Action != pb.Action_Get {
		return nil, ErrWrongAction
	}

	file_type, ok := DataFolders[data.Info.Type]
	if !ok {
		return nil, ErrUnexpectedFileType
	}

	if data.Info.File == "" {
		return nil, ErrEmptyFilename
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s", s.cfg.WorkspacePath, data.Info.User, file_type, data.Info.File)

	file, err := s.openFile(file_path, os.O_RDONLY, 0220)
	if err != nil {
		return nil, err
	}

	read_data := make([]byte, data.Info.GetSize().Chunk)
	n, err := file.ReadAt(read_data, data.Part.Offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return &pb.FilePart{
		Body:   read_data[:n],
		Offset: data.Part.Offset,
		IsLast: err == io.EOF,
	}, nil
}

func (s *DataServer) SaveData(ctx context.Context, data *pb.Data) (*emptypb.Empty, error) {
	file_type, ok := DataFolders[data.Info.Type]
	if !ok {
		return nil, ErrUnexpectedFileType
	}

	if data.Info.File == "" {
		return nil, ErrEmptyFilename
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s.part", s.cfg.WorkspacePath, data.Info.User, file_type, data.Info.File)

	switch data.Action {
	case pb.Action_Create:
		file, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0660)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		s.cache.Push(file_path, file)
		slog.InfoContext(ctx, "Create file - "+file_path)

	case pb.Action_Patch:
		file, err := s.openFile(file_path, os.O_WRONLY, 0440)
		if err != nil {
			return nil, err
		}

		//slog.Info("write to file", slog.Uint64("chunk", data.Info.GetSize().Chunk))

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

func (s *DataServer) GetSum(ctx context.Context, info *pb.DataInfo) (*pb.SHASum, error) {
	file_type, ok := DataFolders[info.Type]
	if !ok {
		return nil, ErrUnexpectedFileType
	}

	if info.File == "" {
		return nil, ErrEmptyFilename
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s", s.cfg.WorkspacePath, info.User, file_type, info.File)

	file, err := s.openFile(file_path, os.O_RDONLY, 0400)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}

	sha := sha256.Sum256(body)
	return &pb.SHASum{
		Sum: sha[:],
	}, nil
}

func (s *DataServer) GetChunkSize(ctx context.Context, info *pb.DataInfo) (*pb.FileSize, error) {
	file_size := info.GetSize().Size
	available_ram := min(s.cfg.Memory.AvailableRAM, freemem.GetAvailableMemory())

	ram_based := available_ram / uint64(len(s.active_files)+1)
	file_based := dataconfig.BASE_CHUNK_SIZE * uint64(math.Log2(float64(file_size)/float64(dataconfig.BASE_CHUNK_SIZE)+1))
	chunk_size := (max(s.cfg.Memory.MinChunkSize, min(min(ram_based, file_based), s.cfg.Memory.MaxChunkSize)) / 4096) * 4096

	return &pb.FileSize{
		Size:  file_size,
		Chunk: chunk_size,
	}, nil
}
