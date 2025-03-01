package create

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeCmd(packageName, name string) error {
	path, err := file.MakeFolder("cmd", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "Dockerfile", modelDockerfile, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "main.go", modelMain, packageName, name)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCmd(packageName string) error {
	path := strs.Format(`./cmd/%s`, packageName)
	_, err := file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = strs.Format(`./internal/services/%s`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = strs.Format(`./internal/pkg/%s`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = strs.Format(`./internal/rest/%s.http`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	return nil
}
