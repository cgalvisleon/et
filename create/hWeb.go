package create

import "github.com/cgalvisleon/et/file"

func MakeWWW(name string) error {
	_, err := file.MakeFolder("www", name)
	if err != nil {
		return err
	}

	return nil
}
