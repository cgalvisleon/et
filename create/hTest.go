package create

import "github.com/cgalvisleon/et/file"

/**
* MakeTest: creates the test folder; name is not used.
* @param name string
* @return error
**/
func MakeTest(name string) error {
	_, err := file.MakeFolder("test")
	if err != nil {
		return err
	}

	return nil
}
