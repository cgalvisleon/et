package create

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
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
	path := et.Format(`./cmd/%s`, packageName)
	_, err := file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = et.Format(`./internal/service/%s`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = et.Format(`./internal/pkg/%s`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = et.Format(`./internal/rest/%s.http`, packageName)
	_, err = file.RemoveFile(path)
	if err != nil {
		return err
	}

	return nil
}
