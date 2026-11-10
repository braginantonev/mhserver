package files

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/braginantonev/mhserver/pkg/contextkeys"
)

func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(contextkeys.USERNAME).(string)
	if !ok {
		return "", false
	}
	return username, true
}

func CompileUserDirectory(username, dir string) (string, error) {
	if !DirIsCorrect(dir) {
		return "", ErrBadDirSyntax
	}

	return filepath.Join(username, dir), nil
}

func CompileUserFilepath(username, dir, file string) (string, error) {
	user_dir, err := CompileUserDirectory(username, dir)
	if err != nil {
		return "", err
	}

	if !FileIsCorrect(file) {
		return "", ErrBadFilenameSyntax
	}

	return filepath.Join(user_dir, file), nil
}

func DirIsCorrect(path string) bool {
	return len(path) != 0 && len(path) < 75 && !strings.Contains(path, "..")
}

func FileIsCorrect(filename string) bool {
	return len(filename) != 0 && len(filename) < 75 && !strings.ContainsRune(filename, '/')
}
