package create

import (
	"github.com/cgalvisleon/elvis/file"
	"github.com/cgalvisleon/elvis/strs"
)

func MakeScripts(name string) error {
	path, err := file.MakeFolder("scripts")
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, strs.Format("%s.http", name), restHttp, name)
	if err != nil {
		return err
	}

	return nil
}
