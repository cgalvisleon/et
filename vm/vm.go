package vm

import (
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
)

type VM struct {
	*Loader `json:"-"`
	Ctx     et.Json       `json:"ctx"`
	vm      *goja.Runtime `json:"-"`
	watch   *file.Watcher `json:"-"`
}

func New(baseDir string) (*VM, error) {
	result := &VM{
		Loader: newLoader(baseDir),
		Ctx:    et.Json{},
	}

	return result, nil
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
		_, err := s.RunFile(s.Main)
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
* RunString
* @params str string
* @return goja.Value, error
**/
func (s *VM) RunString(str string) (goja.Value, error) {
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
* RunFile
* @params path string
* @return goja.Value, error
**/
func (s *VM) RunFile(path string) (goja.Value, error) {
	path = filepath.Join(s.Loader.BaseDir, path)
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := s.RunString(string(data))
	if err != nil {
		return nil, err
	}
	return result, nil
}
