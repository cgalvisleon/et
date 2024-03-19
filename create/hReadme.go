package create

import "github.com/cgalvisleon/elvis/file"

func MakeReadme(packageName string) error {
	_, err := file.MakeFile("", "README.md", modelReadme, packageName)
	if err != nil {
		return err
	}

	return nil
}
