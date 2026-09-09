package create

import (
	"fmt"

	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
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
	path := fmt.Sprintf(`./cmd/%s`, packageName)
	_, err := file.Remove(path)
	if err != nil {
		return err
	}

	path = fmt.Sprintf(`./internal/services/%s`, packageName)
	_, err = file.Remove(path)
	if err != nil {
		return err
	}

	path = fmt.Sprintf(`./pkg/%s`, packageName)
	_, err = file.Remove(path)
	if err != nil {
		return err
	}

	path = fmt.Sprintf(`./scripts/%s.http`, packageName)
	_, err = file.Remove(path)
	if err != nil {
		return err
	}

	return nil
}
