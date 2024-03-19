package create

import "github.com/cgalvisleon/elvis/file"

func MakeWeb(name string) error {
	_, err := file.MakeFolder(name, "web")
	if err != nil {
		return err
	}

	return nil
}
