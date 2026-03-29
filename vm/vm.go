package vm

import (
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type VM struct {
	vm     *goja.Runtime
	ctx    et.Json
	loader *Loader
}

func New(baseDir string) (*VM, error) {
	result := &VM{
		vm:     goja.New(),
		ctx:    et.Json{},
		loader: newLoader(baseDir),
	}
	err := result.wrap()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *VM) wrap() error {
	wrapperRunTime(s)
	wrapperConsole(s)
	wrapperFetch(s)
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
	path = filepath.Join(s.loader.baseDir, path)
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
