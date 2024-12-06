package cmds

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/mistake"
)

func Load(fileName string) (*Stage, error) {
	exist := file.ExistPath(fileName)
	if !exist {
		return nil, mistake.New(MSG_FILE_NOT_FOUND)
	}

	return nil, nil
}
