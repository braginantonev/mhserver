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
	"github.com/braginantonev/mhserver/internal/repository/fileuuidmap"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	pb "github.com/braginantonev/mhserver/proto/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	cfg         dataconfig.DataServiceConfig
	cache       *filecache.FileCache
	activeFiles *fileuuidmap.FileUUIDMap
	sem         chan any
}

func NewDataServer(ctx context.Context, cfg dataconfig.DataServiceConfig) *DataServer {
	return &DataServer{
		cfg:         cfg,
		cache:       filecache.NewFileCache(ctx),
		activeFiles: fileuuidmap.NewFileUUIDMap(ctx),
		sem:         make(chan any, dataconfig.STANDARD_MAX_SAVE_REQUESTS),
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

func (s *DataServer) CreateConnection(ctx context.Context, info *pb.DataInfo) (*pb.Connection, error) {
	if info.Filename == "" {
		return nil, ErrEmptyFilename
	}

	available_ram := min(s.cfg.Memory.AvailableRAM, freemem.GetAvailableMemory())

	ram_based := available_ram / uint64(s.activeFiles.Length())
	file_based := uint64(float64(dataconfig.BASE_CHUNK_SIZE) * math.Log2(float64(info.Size)/float64(dataconfig.BASE_CHUNK_SIZE)+1))

	var chunk_size uint64
	if info.Size < s.cfg.Memory.MinChunkSize {
		chunk_size = info.Size
	} else {
		chunk_size = max(s.cfg.Memory.MinChunkSize, min(min(ram_based, file_based), s.cfg.Memory.MaxChunkSize))
	}

	// Round to RAM page
	if chunk_size > 4096 {
		chunk_size = (chunk_size / 4096) * 4096
	}

	filetype, ok := catalogs[info.Filetype]
	if !ok {
		return nil, ErrUnexpectedFileType
	}

	// "%s%s/%s/%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + file path (with filename)
	file_path := fmt.Sprintf("%s%s/%s/%s", s.cfg.WorkspacePath, info.Username, filetype, info.Filename)

	chunks_count := int(math.Ceil(float64(info.Size) / float64(chunk_size)))

	// Create connection & register file changes
	uuid := s.activeFiles.Add(file_path, fileuuidmap.NewChunksInfo(chunk_size, chunks_count))

	return &pb.Connection{
		UUID:        uuid.String(),
		ChunkSize:   chunk_size,
		ChunksCount: int32(chunks_count),
	}, nil
}

func (s *DataServer) GetData(ctx context.Context, get_chunk *pb.GetChunk) (*pb.FilePart, error) {
	uuid, err := uuid.Parse(get_chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file_info, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrUnexpectedFileChange
	}

	file, err := s.openFile(file_info.GetPath(), os.O_RDONLY, 0440)
	if err != nil {
		return nil, err
	}

	offset := int64(file_info.GetChunkSize()) * int64(get_chunk.ChunkId)

	read_data := make([]byte, file_info.GetChunkSize())
	n, err := file.ReadAt(read_data, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	// The worst thing I've ever written
	if n != 0 && err == io.EOF {
		err = nil
	}

	return &pb.FilePart{
		Chunk:  read_data[:n],
		Offset: offset,
	}, err
}

func (s *DataServer) SaveData(ctx context.Context, save_chunk *pb.SaveChunk) (*emptypb.Empty, error) {
	defer func() {
		<-s.sem
	}()

	s.sem <- struct{}{}

	uuid, err := uuid.Parse(save_chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file_info, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrUnexpectedFileChange
	}

	if len(save_chunk.Data.Chunk) > int(file_info.GetChunkSize()) {
		return nil, ErrIncorrectChunkSize
	}

	file, err := s.openFile(file_info.GetPath()+".part", os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return nil, err
	}

	_, err = file.WriteAt(save_chunk.Data.Chunk, save_chunk.Data.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	err = s.activeFiles.UpdateLoadedChunks(uuid)
	if err == nil {
		return nil, nil
	}

	// Complete save
	if err == fileuuidmap.EOC {
		err := os.Rename(file_info.GetPath()+".part", file_info.GetPath())
		if err != nil {
			return nil, ErrFileNotExist
		}
		slog.InfoContext(ctx, "Rename - "+file_info.GetPath())

		return nil, nil
	}

	// Предположительно, программа никогда не дойдёт до этого момента
	return nil, ErrUnexpectedFileChange
}

func (s *DataServer) GetSum(ctx context.Context, get_chunk *pb.GetChunk) (*pb.SHASum, error) {
	uuid, err := uuid.Parse(get_chunk.UUID)
	if err != nil {
		return nil, ErrBadUUID
	}

	file_info, ok := s.activeFiles.Get(uuid)
	if !ok {
		return nil, ErrUnexpectedFileChange
	}

	file, err := s.openFile(file_info.GetPath(), os.O_RDONLY, 0440)
	if err != nil {
		return nil, err
	}

	body := make([]byte, file_info.GetChunkSize())
	n, err := file.ReadAt(body, int64(file_info.GetChunkSize())*int64(get_chunk.ChunkId))
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}

	// The worst thing I've ever written
	if n != 0 && err == io.EOF {
		err = nil
	}

	sha := sha256.Sum256(body[:n])
	return &pb.SHASum{
		Sum: sha[:],
	}, err
}
