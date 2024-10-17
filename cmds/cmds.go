package cmds

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
)

func Load(fileName string) (*Stage, error) {
	exist := file.ExistPath(fileName)
	if !exist {
		return nil, logs.NewError(MSG_FILE_NOT_FOUND)
	}

	return nil, nil
}
