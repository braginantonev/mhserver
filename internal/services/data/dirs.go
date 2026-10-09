package data

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/pkg/contextkeys"
)

func GetUsernameFromContext(ctx context.Context) (string, error) {
	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		return "", errors.New("failed get username from context")
	}
	return username, nil
}

func CompileUserDirectory(username, dir string, target_service services.ServiceName) (string, error) {
	if !DirIsCorrect(dir) {
		return "", ErrBadDirSyntax
	}

	// files service keep files in uspace
	if target_service == SERVICE_NAME {
		return filepath.Join(username, dir), nil
	}

	// another services keep files in protected dirs
	return filepath.Join(username, string(target_service), dir), nil
}

func CompileUserFilepath(username, dir, file string, target_service services.ServiceName) (string, error) {
	user_dir, err := CompileUserDirectory(username, dir, target_service)
	if err != nil {
		return "", err
	}

	if !FileIsCorrect(file) {
		return "", ErrBadFilenameSyntax
	}

	return filepath.Join(user_dir, file), nil
}

func DirIsCorrect(path string) bool {
	return len(path) != 0 && !strings.Contains(path, "..")
}

func FileIsCorrect(filename string) bool {
	return len(filename) != 0 && !strings.ContainsRune(filename, '/')
}
