package create

import "github.com/cgalvisleon/et/file"

func MakeReadme(packageName string) error {
	_, err := file.Make("./", "README.md", modelReadme, packageName)
	if err != nil {
		return err
	}

	return nil
}
