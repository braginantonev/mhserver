package data

import (
	"fmt"
	"os"
)

func GenerateUserFolders(path string, folders ...string) error {
	var err error
	for _, folder := range folders {
		err = os.MkdirAll(fmt.Sprintf("%s/%s", path, folder), 0660)
		if err != nil {
			return err
		}
	}
	return nil
}
