package create

import "github.com/cgalvisleon/et/file"

func MakeInternal(packageName, name string) error {
	_, err := file.MakeFolder("internal", "data")
	if err != nil {
		return err
	}

	path, err := file.MakeFolder("internal", "service", name)
	if err != nil {
		return err
	}

	_, err = file.Make(path, "service.go", modelService, packageName, name)
	if err != nil {
		return err
	}

	path, err = file.MakeFolder("internal", "service", name, "v1")
	if err != nil {
		return err
	}

	_, err = file.Make(path, "api.go", modelApi, packageName, name)
	if err != nil {
		return err
	}

	return nil
}
