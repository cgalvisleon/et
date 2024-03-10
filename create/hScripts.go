package create

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
)

func MakeScripts(name string) error {
	path, err := file.MakeFolder("scripts")
	if err != nil {
		return err
	}

	_, err = file.Make(path, et.Format("%s.http", name), restHttp, name)
	if err != nil {
		return err
	}

	return nil
}
