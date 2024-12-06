package cmds

import (
	"errors"

	"github.com/cgalvisleon/et/file"
)

func Load(fileName string) (*Stage, error) {
	exist := file.ExistPath(fileName)
	if !exist {
		return nil, errors.New(MSG_FILE_NOT_FOUND)
	}

	return nil, nil
}
