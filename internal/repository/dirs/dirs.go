package dirs

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/braginantonev/mhserver/internal/config"
)

var (
	ErrBadDirSyntax error = errors.New("directory have bad syntax")
)

func GetDataPath(workspace_path, user, req_dir string, service config.ServiceName) (string, error) {
	if len(req_dir) == 0 {
		return "", ErrBadDirSyntax
	}

	if req_dir[0] != '/' || strings.Contains(req_dir, "..") {
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
