package create

import "github.com/cgalvisleon/et/file"

func MakeProject(name string) error {
	_, err := file.MakeFolder(name)
	if err != nil {
		return err
	}

	return nil
}
