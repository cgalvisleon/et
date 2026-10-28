package create

import "github.com/cgalvisleon/et/file"

/**
* MakeWWW: creates the www/<name> folder.
* @param name string
* @return error
**/
func MakeWWW(name string) error {
	_, err := file.MakeFolder("www", name)
	if err != nil {
		return err
	}

	return nil
}
