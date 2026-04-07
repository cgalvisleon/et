package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
)

/**
* Build
* @param store Store, part Part
* @return *VM, error
**/
func Build(store Store, part Part) (*VM, error) {
	if store == nil {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "store")
	}

	absPath, err := filepath.Abs("")
	if err != nil {
		return nil, err
	}

	pkgFile := filepath.Join(absPath, "package.json")
	if !exists(pkgFile) {
		return nil, fmt.Errorf("package.json not found")
	}

	result := &VM{
		Loader: newLoader(""),
		Ctx:    et.Json{},
	}
	data, _ := os.ReadFile(pkgFile)
	err = json.Unmarshal(data, &result.Pkg)
	if err != nil {
		return nil, err
	}

	result.store = store
	result.mode = Building
	err = result.init()
	if err != nil {
		return nil, err
	}

	if result.store != nil {
		err := result.store.Connected()
		if err != nil {
			return nil, err
		}
	}

	_, err = result.BumpVersion(part)
	if err != nil {
		return nil, err
	}

	for module, path := range result.Pkg.Scripts {
		inf := file.ExistPath(path)
		if inf.IsDir {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		id := fmt.Sprintf("pkg:%s:%s:%s", result.Name, result.Version, module)
		result.store.Set(id, &Module{
			ID:      id,
			Scripts: string(data),
		})
	}

	return result, nil
}
