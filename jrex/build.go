package jrex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
)

type Module struct {
	ID      string `json:"id"`
	Scripts string `json:"scripts"`
}

/**
* Build
* @param part Part
* @return error
**/
func (s *Jrex) Build(part Part) error {
	if s.mode != Develop {
		return errors.New("build is only available in develop mode")
	}

	if s.store == nil {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "store")
	}

	if !exists(s.pkgFile) {
		return errors.New("package.json not found")
	}

	pkgData, _ := os.ReadFile(s.pkgFile)
	err := json.Unmarshal(pkgData, &s.Pkg)
	if err != nil {
		return err
	}

	_, err = s.BumpVersion(part)
	if err != nil {
		return err
	}

	err = s.uppToStore()
	if err != nil {
		return err
	}

	for module, path := range s.Pkg.Modules {
		inf := file.ExistPath(path)
		if inf.IsDir {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		id := fmt.Sprintf("pkg:%s:%s:%s", s.Name, module, s.Version)
		s.store.SetModule(id, &Module{
			ID:      id,
			Scripts: string(data),
		})
	}

	return nil
}
