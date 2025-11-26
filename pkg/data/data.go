package data

import "os"

func GenerateUserFolders(workspace_path string, folders ...string) error {
	var err error
	for _, folder := range folders {
		err = os.Mkdir(workspace_path+folder, 0660)
		if err != nil {
			return err
		}
	}
	return nil
}
