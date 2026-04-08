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

type Module struct {
	ID      string `json:"id"`
	Scripts string `json:"scripts"`
}

type VM struct {
	*Loader `json:"-"`
	Ctx     et.Json       `json:"ctx"`
	vm      *goja.Runtime `json:"-"`
	watch   *file.Watcher `json:"-"`
}

/**
* New
* @param baseDir, name, version string
* @return *VM, error
**/
func New(name string) (*VM, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	result := &VM{
		Loader: newLoader(name),
		Ctx:    et.Json{},
	}
	return result, nil
}

/**
* RunDev
* @param baseDir string
* @return error
**/
func (s *VM) RunDev(baseDir string) error {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	s.BaseDir = absPath
	s.mode = Develop
	err = s.init()
	if err != nil {
		return err
	}

	_, err = s.RunByFile(s.Main)
	if err != nil {
		return err
	}

	err = s.HotReload()
	if err != nil {
		return err
	}

	return nil
}

/**
* RunProd
* @param store Store
* @return error
**/
func (s *VM) RunProd(store Store) error {
	if store == nil {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "store")
	}

	s.store = store
	s.mode = Production
	err := s.init()
	if err != nil {
		return err
	}

	_, err = s.RunBySource(s.Main)
	if err != nil {
		return err
	}

	return nil
}

/**
* uppToStore
* @return error
**/
func (s *VM) uppToStore() error {
	id := fmt.Sprintf("pkg:%s:%s", s.Name, s.Version)
	return s.set(id, s)
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
* SetModel
* @params module string, path string
* @return error
**/
func (s *VM) SetModel(module string, path string) error {
	_, ok := s.Models[module]
	s.Models[module] = path
	if !ok {
		return s.save()
	}
	return nil
}

/**
* SetDescription
* @params description string
* @return error
**/
func (s *VM) SetDescription(description string) error {
	s.Description = description
	return s.save()
}

/**
* SetAuthor
* @params author string
* @return error
**/
func (s *VM) SetAuthor(author string) error {
	s.Author = author
	return s.save()
}

/**
* SetLicense
* @params license string
* @return error
**/
func (s *VM) SetLicense(license string) error {
	s.License = license
	return s.save()
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
	var scr Module
	exists, err := s.get(path, &scr)
	if err != nil {
		panic(s.Error(err))
	}

	if !exists {
		panic(s.Error(fmt.Errorf("script not found: %s", path)))
	}
	return s.Run(scr.Scripts)
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
