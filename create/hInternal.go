package create

import "github.com/cgalvisleon/elvis/file"

func MakeInternal(packageName, name string) error {
	_, err := file.MakeFolder("internal", "data")
	if err != nil {
		return err
	}

	path, err := file.MakeFolder("internal", "service", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "service.go", modelService, packageName, name)
	if err != nil {
		return err
	}

	path, err = file.MakeFolder("internal", "service", name, "v1")
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "api.go", modelApi, packageName, name)
	if err != nil {
		return err
	}

	return nil
}
