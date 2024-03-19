package create

import "github.com/cgalvisleon/et/file"

func MakeEnv(packageName string) error {
	_, err := file.MakeFile("", ".env", modelEnvar, packageName)
	if err != nil {
		return err
	}

	return nil
}
