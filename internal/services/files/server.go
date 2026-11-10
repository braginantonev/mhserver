package files

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/braginantonev/mhserver/internal/repository"
	"github.com/braginantonev/mhserver/internal/repository/freemem"
	"github.com/braginantonev/mhserver/pkg/contextkeys"
	pb "github.com/braginantonev/mhserver/proto/gen/files"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FilesServer struct {
	pb.FilesServiceServer

	cfg DataServiceConfig
	sem repository.Semaphore

	// Ensured users - it's users, whish user dir create in every uspace.
	// To avoid checking every rpc, we keep them in array.
	// Also, server is home, so users can't be more than 10 e.g., so I use array instead hash map
	// Update this, if server will be support deleting the users.
	ensuredUsers []string
}

func NewFilesServer(ctx context.Context, cfg DataServiceConfig) *FilesServer {
	return &FilesServer{
		cfg:          cfg,
		sem:          repository.NewSemaphore(SEMAPHORE_SIZE),
		ensuredUsers: make([]string, 0, 10),
	}
}

// Exist user dir or not. If not exist - create user dir in all user spaces.
func (s *FilesServer) ensureUserDir(ctx context.Context, username string) error {
	if slices.Contains(s.ensuredUsers, username) {
		return nil
	}

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

	s.ensuredUsers = append(s.ensuredUsers, username)
	return nil
}

func (s *FilesServer) findFileInUserSpaces(ctx context.Context, file string, flag int) (*os.File, error) {
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

func (s *FilesServer) acceptModifyDir(ctx context.Context, target_dir string) bool {
	// split to grab dir in root
	target_dir = strings.SplitN(strings.TrimPrefix(target_dir, "/"), "/", 2)[0]

	if slices.Contains(s.cfg.ServiceDirs, target_dir) {
		is_internal, ok := ctx.Value(contextkeys.InternalRequest).(bool)
		return is_internal && ok
	}

	// if target dir is not service dir - in any case we accepting modification
	return true
}

func (s *FilesServer) SaveFile(stream pb.FilesService_SaveFileServer) error {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(stream.Context())
	if !ok {
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

	target_dir := meta.Dir.GetValue()

	if !s.acceptModifyDir(stream.Context(), target_dir) {
		return ErrModifyProtectedDir
	}

	user_file, err := CompileUserFilepath(username, target_dir, meta.Name)
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

func (s *FilesServer) ReadFile(req *pb.RequiredFile, stream pb.FilesService_ReadFileServer) error {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(stream.Context())
	if !ok {
		slog.ErrorContext(stream.Context(), "failed get username")
		return ErrInternal
	}

	if err := s.ensureUserDir(stream.Context(), username); err != nil {
		return err
	}

	user_file, err := CompileUserFilepath(username, req.Dir.GetValue(), req.Name)
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
	data := make([]byte, s.cfg.ChunkSize)

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

			if err = stream.Send(&pb.Chunk{
				Offset: i * s.cfg.ChunkSize,
				Data:   data[:n],
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *FilesServer) GetSum(ctx context.Context, req *pb.RequiredFile) (*pb.SHASum, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_file, err := CompileUserFilepath(username, req.Dir.GetValue(), req.Name)
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

func (s *FilesServer) GetAvailableDiskSpace(ctx context.Context, _ *emptypb.Empty) (*pb.Size, error) {
	s.sem.Acquire()
	defer s.sem.Release()

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

func (s *FilesServer) GetFiles(ctx context.Context, dir *pb.Directory) (*pb.FilesList, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, dir.GetValue())
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

func (s *FilesServer) CreateDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	target_dir := dir.GetValue()

	if !s.acceptModifyDir(ctx, target_dir) {
		return nil, ErrModifyProtectedDir
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, target_dir)
	if err != nil {
		return nil, err
	}

	// We use Stat() to check exist dir or not. Of course os.MkdirAll also use Stat() which we use pass already existed dir, but he don't check if this existed dir is not dir.
	// If we try to MkdirAll on existed `test.json` (for example), we get error: `syscall.ENOTDIR`. Yea, we can use errors.Is() and check that, but if this error
	// we get in 2 uspace e.g.? We create dirs in 0 and 1 uspace's, but get error in 2. It's a break a rule: dir in every uspace.
	// Mb it's wrong. I don't like that double Stat() usage
	for _, usp := range s.cfg.UserSpaces {
		fi, err := os.Stat(filepath.Join(s.cfg.WorkspacePath, usp, user_dir))
		if err != nil {
			if !os.IsNotExist(err) {
				slog.ErrorContext(ctx, "failed get dir stat", slog.Any("error", err))
				return nil, ErrInternal
			}
		} else {
			if fi.IsDir() {
				return nil, ErrDirAlreadyExist
			} else {
				return nil, ErrBadDirSyntax
			}
		}
	}

	for _, usp := range s.cfg.UserSpaces {
		if err := os.MkdirAll(filepath.Join(s.cfg.WorkspacePath, usp, user_dir), 0770); err != nil {
			slog.ErrorContext(ctx, "failed create user directory", slog.Any("err", err))
			return nil, ErrInternal
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *FilesServer) RemoveDir(ctx context.Context, dir *pb.Directory) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	target_dir := dir.GetValue()

	if !s.acceptModifyDir(ctx, target_dir) {
		return nil, ErrModifyProtectedDir
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_dir, err := CompileUserDirectory(username, target_dir)
	if err != nil {
		return nil, err
	}

	for _, usp := range s.cfg.UserSpaces {
		if err := os.RemoveAll(filepath.Join(s.cfg.WorkspacePath, usp, user_dir)); err != nil {
			slog.ErrorContext(ctx, "failed remove user direction", slog.Any("err", err))
			return nil, ErrInternal
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *FilesServer) RemoveFile(ctx context.Context, req *pb.RequiredFile) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	target_dir := req.Dir.GetValue()

	if !s.acceptModifyDir(ctx, target_dir) {
		return nil, ErrModifyProtectedDir
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	user_file, err := CompileUserFilepath(username, req.GetDir().Value, req.GetName())
	if err != nil {
		return nil, err
	}

	var removed bool
	for _, usp := range s.cfg.UserSpaces {
		err := os.Remove(filepath.Join(s.cfg.WorkspacePath, usp, user_file))
		if err == nil {
			removed = true
			break // one uspace - one file
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

	return &emptypb.Empty{}, nil
}

func (s *FilesServer) RenameDir(ctx context.Context, req *pb.RenameDirectoryRequest) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	old_dir := req.Old.GetValue()
	if !s.acceptModifyDir(ctx, old_dir) {
		return nil, ErrModifyProtectedDir
	}

	// we also check a new dir, because os.Rename don't cause an error, if old and new paths in different dirs
	// user can set new_dir in protected dir, what not good of course
	new_dir := req.New.GetValue()
	if !s.acceptModifyDir(ctx, new_dir) {
		return nil, ErrModifyProtectedDir
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	target, err := CompileUserDirectory(username, old_dir)
	if err != nil {
		return nil, err
	}

	new, err := CompileUserDirectory(username, new_dir)
	if err != nil {
		return nil, err
	}

	var renamed bool
	for _, usp := range s.cfg.UserSpaces {
		if err = os.Rename(filepath.Join(s.cfg.WorkspacePath, usp, target), filepath.Join(s.cfg.WorkspacePath, usp, new)); err == nil {
			renamed = true
			// but we continue check next userspaces
		} else {
			if os.IsNotExist(err) {
				continue
			}

			slog.ErrorContext(ctx, "failed rename dir", slog.Any("error", err))
			return nil, ErrInternal
		}
	}

	if !renamed {
		return nil, ErrDirNotFound
	}

	return &emptypb.Empty{}, nil
}

func (s *FilesServer) RenameFile(ctx context.Context, req *pb.RenameFileRequest) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "failed get username")
		return nil, ErrInternal
	}

	// besides RenameDir() we don't check new_dir, because we use only old dir
	// simple, we just skip target directory and use old
	target_dir := req.Old.GetDir().GetValue()
	if !s.acceptModifyDir(ctx, target_dir) {
		return nil, ErrModifyProtectedDir
	}

	if err := s.ensureUserDir(ctx, username); err != nil {
		return nil, err
	}

	target, err := CompileUserFilepath(username, target_dir, req.Old.GetName())
	if err != nil {
		return nil, err
	}

	new, err := CompileUserFilepath(username, target_dir, req.New.GetName())
	if err != nil {
		return nil, err
	}

	var renamed bool
	for _, usp := range s.cfg.UserSpaces {
		if err = os.Rename(filepath.Join(s.cfg.WorkspacePath, usp, target), filepath.Join(s.cfg.WorkspacePath, usp, new)); err == nil {
			renamed = true
			break // one file - one uspace
		} else {
			if os.IsNotExist(err) {
				continue
			}

			slog.ErrorContext(ctx, "failed rename file", slog.Any("error", err))
			return nil, ErrInternal
		}
	}

	if !renamed {
		return nil, ErrFileNotExist
	}

	return &emptypb.Empty{}, nil
}
