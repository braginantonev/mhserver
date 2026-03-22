package dirs

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrBadDirSyntax error = errors.New("directory have bad syntax")
)

/*
directoryRegexp check:
  - dir start with /
  - dir name contains only letters, numbers and underscores, whitespaces and hyphens
  - dir name can't be started and ended with space, underscore and hyphen
  - dir name have one letter or number
  - dir name can have dot in start (hidden folder)
  - dir name can't be only one dot
  - dir end with /, if dir not root
*/
var directoryRegexp = regexp.MustCompile(`^\/(\.?[\p{L}\p{N}]+([ _-]+[\p{L}\p{N}]+)*\/)*$`)

func GetDataPath(workspace_path, user, service, req_dir string) (string, error) {
	if !directoryRegexp.MatchString(req_dir) {
		return "", ErrBadDirSyntax
	}

	// "%s%s/%s%s" -> "/home/srv/.mhserver/" + username + file type (File, Image, Music etc) + directory
	return fmt.Sprintf("%s%s/%s%s", workspace_path, user, service, req_dir), nil
}
