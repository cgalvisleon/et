package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
)

type VM struct {
	*Loader `json:"-"`
	Ctx     et.Json       `json:"ctx"`
	vm      *goja.Runtime `json:"-"`
	watch   *file.Watcher `json:"-"`
}

/**
* Dev
* @param baseDir, name, version string
* @return *VM, error
**/
func Dev(baseDir, name, version string) (*VM, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(version, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "version")
	}

	result := &VM{
		Loader: newLoader(baseDir, name, version),
		Ctx:    et.Json{},
	}
	result.mode = Develop
	err := result.init()
	if err != nil {
		return nil, err
	}

	_, err = result.RunByFile(result.Main)
	if err != nil {
		return nil, err
	}

	err = result.HotReload()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Prod
* @param name, version string, store Store
* @return *VM, error
**/
func Prod(name, version string, store Store) (*VM, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(version, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "version")
	}
	if store == nil {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "store")
	}

	result := &VM{
		Loader: newLoader("", name, version),
		Ctx:    et.Json{},
	}
	result.store = store
	result.mode = Production
	err := result.init()
	if err != nil {
		return nil, err
	}

	_, err = result.RunBySource(result.Main)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Build
* @param baseDir, name, version string, store Store
* @return *VM, error
**/
func Build(baseDir, name, version string, store Store) (*VM, error) {
	result, err := Dev(baseDir, name, version)
	if err != nil {
		return nil, err
	}
	result.store = store
	result.mode = Building
	return result, nil
}

/**
* Value
* @param value interface{}
* @return goja.Value
**/
func (s *VM) Value(value interface{}) goja.Value {
	return s.vm.ToValue(value)
}

/**
* Error
* @param err error
* @return *goja.Object
**/
func (s *VM) Error(err error) *goja.Object {
	return s.vm.NewGoError(err)
}

/**
* Set
* @params name string, value interface{}
* @return error
**/
func (s *VM) Set(name string, value interface{}) error {
	return s.vm.Set(name, value)
}

/**
* Run
* @params str string
* @return goja.Value, error
**/
func (s *VM) Run(str string) (goja.Value, error) {
	s.vm = goja.New()
	wrap(s)

	_, err := s.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	result, err := s.vm.RunString(str)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* RunByFile
* @params path string
* @return goja.Value, error
**/
func (s *VM) RunByFile(path string) (goja.Value, error) {
	path = filepath.Join(s.Loader.BaseDir, path)
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := s.Run(string(data))
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* RunBySource
* @param path string
* @return (goja.Value, error)
**/
func (s *VM) RunBySource(path string) (goja.Value, error) {
	return s.RunByFile(path)
}

/**
* HotReload
* @return error
**/
func (s *VM) HotReload() error {
	watch, err := file.NewWatcher(s.BaseDir)
	if err != nil {
		return err
	}
	s.watch = watch
	err = s.watch.OnReload(func(info file.FileInfo, event fsnotify.Event) {
		_, err := s.RunByFile(s.Main)
		if err != nil {
			logs.Error(err)
		} else {
			logs.Log("Hot Reloaded:", s.Ctx.ToString())
		}
	}).Load()
	if err != nil {
		return err
	}
	return nil
}
