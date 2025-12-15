package data

import (
	"fmt"
	"os"

	pb "github.com/braginantonev/mhserver/proto/data"
)

var (
	// Так себе, но ладно. Потом что-нибудь придумаю
	DataFolders = map[pb.DataType]string{
		pb.DataType_File:  "files",
		pb.DataType_Image: "images",
		pb.DataType_Music: "music",
	}
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
