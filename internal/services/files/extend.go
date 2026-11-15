package files

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Instead `os.MkdirAll` this func not check exist file or dir or not. Check this before.
// So this function not using os.Stat()
func mkdirAllTrust(uspace, target string, perm os.FileMode) error {
	dirs := strings.SplitSeq(strings.TrimPrefix(target, "/"), "/")
	target = uspace
	for dir := range dirs {
		target = filepath.Join(target, dir)
		if err := os.Mkdir(target, perm); err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// Used unlink from syscall package instead `os.Remove`.
// `os.Remove` remove and dir and file, it's a bad if I know what I wan't to remove
// For example I use removeFile in RemoveFile rpc of course and if set filename in req like existed dir -> rpc return nil
// Yea, RemoveFile remove dir and return nil. Of course it's bad
func removeFile(file string) error {
	for {
		err := syscall.Unlink(file)
		if err == syscall.EINTR {
			continue
		}
		return err
	}
}
