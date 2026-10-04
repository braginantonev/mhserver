package data

import (
	"path/filepath"
	"strings"

	"github.com/braginantonev/mhserver/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrBadDirSyntax error = status.Error(codes.InvalidArgument, "directory have bad syntax")

func GetDataPath(workspace_path, uspace, uname, target_dir string, target_service services.ServiceName) (string, error) {
	if !DirIsCorrect(target_dir) {
		return "", ErrBadDirSyntax
	}

	// files service keep files in uspace
	if target_service == SERVICE_NAME {
		return filepath.Join(workspace_path, uspace, uname, target_dir), nil
	}

	// another services keep files in protected dirs
	return filepath.Join(workspace_path, uspace, uname, string(target_service), target_dir), nil
}

func DirIsCorrect(path string) bool {
	return len(path) != 0 && !strings.Contains(path, "..")
}

func FileIsCorrect(filename string) bool {
	return len(filename) != 0 && !strings.ContainsRune(filename, '/')
}
