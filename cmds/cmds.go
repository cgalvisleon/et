package cmds

import (
	"fmt"

	"github.com/cgalvisleon/et/file"
)

func Load(fileName string) (*Stage, error) {
	info := file.ExistPath(fileName)
	if info.Error != nil {
		return nil, info.Error
	} else if !info.Exist {
		return nil, fmt.Errorf(MSG_FILE_NOT_FOUND)
	}

	return nil, nil
}
