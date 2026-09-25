package dirs

import (
	"fmt"
	"os"
	"strings"

	"github.com/braginantonev/mhserver/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrBadDirSyntax error = status.Error(codes.InvalidArgument, "directory have bad syntax")
)

func DirIsCorrect(path string) bool {
	return len(path) != 0 && path[0] == '/' && !strings.Contains(path, "..")
}

func FileIsCorrect(filename string) bool {
	return len(filename) != 0 && !strings.ContainsRune(filename, '/')
}

func GetDataPath(workspace_path, user, req_dir string, service services.ServiceName) (string, error) {
	if !DirIsCorrect(req_dir) {
		return "", ErrBadDirSyntax
	}

	if req_dir[len(req_dir)-1] != '/' {
		req_dir += "/"
	}

	// "%s%s/%s%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + directory
	return fmt.Sprintf("%s%s/%s%s", workspace_path, user, service, req_dir), nil
}

func GenerateUserFolders(workspace_path, user string, folders ...string) error {
	// When I just started to work with the server I want create a microservice architecture,
	// but that's idea not liked me now.
	// So I use this function to create user folders to monolith arch.
	for _, folder := range folders {
		err := os.MkdirAll(fmt.Sprintf("%s%s/%s", workspace_path, user, folder), 0660)
		if err != nil {
			return err
		}
	}
	return nil
}
