package data

import (
	"context"
	"crypto/sha256"
	"errors"
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
	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServer struct {
	pb.DataServiceServer
	services.Service

	cfg DataServiceConfig
	sem repository.Semaphore
}

func NewDataServer(ctx context.Context, cfg DataServiceConfig) *DataServer {
	return &DataServer{
		cfg: cfg,
		sem: repository.NewSemaphore(SEMAPHORE_SIZE),
	}
}

// Exist user dir or not. If not exist - create user dir in all user spaces.
// Of course, using a cache for this method - best idea. But this is not a big server, so I'm to lazy to do it.
func (s *DataServer) ensureUserDir(ctx context.Context, username string) error {
	var needPreparingUspace []string

	// we check all of uspace's, because new uspace's don't have an user dir.
	for _, usp := range s.cfg.UserSpaces {
		if _, err := os.Stat(filepath.Join(s.cfg.WorkspacePath, usp, username)); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				slog.ErrorContext(ctx, "failed get stat of user dir", slog.Any("error", err))
				return ErrInternal
			}
			needPreparingUspace = append(needPreparingUspace, usp)
		}
	}

	for _, usp := range needPreparingUspace {
		if err := os.Mkdir(filepath.Join(s.cfg.WorkspacePath, usp, username), 0770); err != nil && !errors.Is(err, os.ErrExist) {
			slog.ErrorContext(ctx, "failed create user dir", slog.Any("error", err))
			return ErrInternal
		}
	}

	return nil
}

func (s *DataServer) findFileInUserSpaces(ctx context.Context, file string, flag int) (*os.File, error) {
	for _, uspace := range s.cfg.UserSpaces {
		file, err := os.OpenFile(filepath.Join(s.cfg.WorkspacePath, uspace, file), flag, 0660)
		if err == nil {
			return file, nil
		}

		if !os.IsNotExist(err) {
			slog.ErrorContext(ctx, "failed open file", slog.Any("error", err))
			return nil, ErrInternal
		}
	}

	return nil, ErrFileNotExist
}

func (s *DataServer) SaveFile(stream pb.DataService_SaveFileServer) error {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(stream.Context())
	if err != nil {
		slog.ErrorContext(stream.Context(), "failed get username")
		return ErrInternal
	}

	if err := s.ensureUserDir(stream.Context(), username); err != nil {
		return err
	}

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

	user_file, err := CompileUserFilepath(username, meta.Dir.GetValue(), meta.Name, SERVICE_NAME)
	if err != nil {
		return err
	}

	var file *os.File
	var file_size uint64
	if meta.NewSize == nil { // update file, no create. We search file in uspace's
		file, err = s.findFileInUserSpaces(stream.Context(), user_file, os.O_WRONLY)
		if err != nil {
			return err
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
				slog.ErrorContext(stream.Context(), "failed get available disk space in uspace", slog.Any("error", err))
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

		file, err = os.OpenFile(filepath.Join(s.cfg.WorkspacePath, available_uspace, user_file), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrDirNotFound
			}

			slog.ErrorContext(stream.Context(), "failed create file", slog.Any("error", err))
			return ErrInternal
		}

		if err = file.Truncate(int64(*meta.NewSize)); err != nil {
			slog.ErrorContext(stream.Context(), "failed truncate file", slog.Any("error", err))
			return ErrInternal
		}

		file_size = *meta.NewSize
	}
	defer func() { _ = file.Close() }()

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

func (s *DataServer) ReadFile(req *pb.RequiredFile, stream pb.DataService_ReadFileServer) error {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(stream.Context())
	if err != nil {
		slog.ErrorContext(stream.Context(), "failed get username")
		return ErrInternal
	}

	if err := s.ensureUserDir(stream.Context(), username); err != nil {
		return err
	}

	user_file, err := CompileUserFilepath(username, req.Dir.GetValue(), req.Name, SERVICE_NAME)
	if err != nil {
		return err
	}

	file, err := s.findFileInUserSpaces(stream.Context(), user_file, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	file_stat, err := file.Stat()
	if err != nil {
		slog.ErrorContext(stream.Context(), "failed get stat of exist file", slog.Any("error", err))
		return ErrInternal
	}

	chunks_count := uint64(math.Ceil(float64(file_stat.Size()) / float64(s.cfg.ChunkSize)))
	data := make([]byte, 0, s.cfg.ChunkSize)

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
				Offset: i * s.cfg.ChunkSize,
				Data:   data[:n],
			})
		}
	}

	return nil
}

func (s *DataServer) GetSum(ctx context.Context, req *pb.RequiredFile) (*pb.SHASum, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_file, err := CompileUserFilepath(username, req.Dir.GetValue(), req.Name, SERVICE_NAME)
	if err != nil {
		return nil, err
	}

	file, err := s.findFileInUserSpaces(ctx, user_file, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		slog.ErrorContext(ctx, "failed copy file for sum calc", slog.Any("error", err))
		return nil, ErrInternal
	}

	return &pb.SHASum{Value: hash.Sum(nil)[:]}, nil
}

func (s *DataServer) GetAvailableDiskSpace(ctx context.Context, _ *emptypb.Empty) (*pb.Size, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		slog.ErrorContext(ctx, "failed get username from context", slog.Any("got", ctx.Value(contextkeys.USERNAME)))
		return nil, ErrInternal
	}

	var total, buff uint64
	for _, usp := range s.cfg.UserSpaces {
		buff, _ = freemem.GetAvailableDiskSpace(filepath.Join(s.cfg.WorkspacePath, usp, username))
		total += buff
	}

	return &pb.Size{Value: total}, nil
}

func (s *DataServer) GetFiles(ctx context.Context, dir *pb.Directory) (*pb.FilesList, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, dir.GetValue(), SERVICE_NAME)
	if err != nil {
		return nil, err
	}

	var total_files int
	entries := make([][]os.DirEntry, 0, len(s.cfg.UserSpaces))
	for _, usp := range s.cfg.UserSpaces {
		files, err := os.ReadDir(filepath.Join(s.cfg.WorkspacePath, usp, user_dir))
		if err != nil {
			if !os.IsNotExist(err) {
				slog.ErrorContext(ctx, "failed read dir", slog.Any("error", err))
				return nil, ErrInternal
			}
			continue
		}
		total_files += len(files)
		entries = append(entries, files)
	}

	if len(entries) == 0 {
		return nil, ErrDirNotFound
	}

	list := &pb.FilesList{
		Value: make([]*pb.FileInfo, 0, total_files),
	}

	for _, entry := range entries {
		for _, file := range entry {
			ret := &pb.FileInfo{
				Name:  file.Name(),
				IsDir: file.IsDir(),
			}

			info, err := file.Info()
			if err != nil {
				continue
			}

			ret.Size = uint64(info.Size())
			ret.ModTime = uint64(info.ModTime().Unix())
			list.Value = append(list.Value, ret)
		}
	}

	return list, nil
}

func (s *DataServer) CreateDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, dir.GetValue(), SERVICE_NAME)
	if err != nil {
		return nil, err
	}

	for _, usp := range s.cfg.UserSpaces {
		if err := os.MkdirAll(filepath.Join(s.cfg.WorkspacePath, usp, user_dir), 0770); err != nil {
			if os.IsExist(err) {
				// we return value here, because if dir already exist - he also exist in other uspace's
				return nil, ErrDirAlreadyExist
			}

			slog.ErrorContext(ctx, "failed create user direction", slog.Any("err", err))
			return nil, ErrInternal
		}
	}

	return nil, nil
}

func (s *DataServer) RemoveDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, dir.GetValue(), SERVICE_NAME)
	if err != nil {
		return nil, err
	}

	for _, usp := range s.cfg.UserSpaces {
		if err := os.RemoveAll(filepath.Join(s.cfg.WorkspacePath, usp, user_dir)); err != nil {
			slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
			return nil, ErrInternal
		}
	}

	return nil, nil
}

func (s *DataServer) RemoveFile(ctx context.Context, req *pb.RequiredFile) (*emptypb.Empty, error) {
	defer s.sem.Release()
	s.sem.Acquire()

	username, err := GetUsernameFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_file, err := CompileUserFilepath(username, req.GetDir().Value, req.GetName(), SERVICE_NAME)
	if err != nil {
		return nil, err
	}

	var removed bool
	for _, usp := range s.cfg.UserSpaces {
		err := os.Remove(filepath.Join(s.cfg.WorkspacePath, usp, user_file))
		if err == nil {
			removed = true
			break
		}

		if os.IsNotExist(err) {
			continue
		}

		slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
		return nil, ErrInternal
	}

	if !removed {
		return nil, ErrFileNotExist
	}

	return nil, nil
}
