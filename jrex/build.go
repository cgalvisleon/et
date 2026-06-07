package jrex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	if s.store == nil {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "store")
	}

	absPath, err := filepath.Abs("")
	if err != nil {
		return err
	}

	pkgFile := filepath.Join(absPath, "package.json")
	if !exists(pkgFile) {
		return fmt.Errorf("package.json not found")
	}

	pkgData, _ := os.ReadFile(pkgFile)
	err = json.Unmarshal(pkgData, &s.Pkg)
	if err != nil {
		return err
	}

	s.mode = Building
	err = s.init()
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
