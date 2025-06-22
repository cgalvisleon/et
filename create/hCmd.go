package create

import (
	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeCmd(packageName, name string) error {
	path, err := file.MakeFolder("cmd", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "Dockerfile", template.ModelDockerfile, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "main.go", template.ModelMain, packageName, name)
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
