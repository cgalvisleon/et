package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
)

/**
* Build
* @param store Store, part Part
* @return *VM, error
**/
func (s *VM) Build(store Store, part Part) error {
	if store == nil {
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

	s.store = store
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
		s.store.Set(id, &Module{
			ID:      id,
			Scripts: string(data),
		})
	}

	return nil
}
