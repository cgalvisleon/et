package create

import "github.com/cgalvisleon/et/file"

func MakeInternal(packageName, name, schema string) error {
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

	if len(schema) > 0 {
		_, err = file.MakeFile(path, "api.go", modelDbApi, packageName, name)
		if err != nil {
			return err
		}
	} else {
		_, err = file.MakeFile(path, "api.go", modelApi, packageName, name)
		if err != nil {
			return err
		}
	}

	return nil
}
