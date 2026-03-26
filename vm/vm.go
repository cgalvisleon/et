package vm

import (
	"os"

	"github.com/dop251/goja"
)

type VM struct {
	vm     *goja.Runtime
	loader *Loader
}

func New(baseDir string) (*VM, error) {
	result := &VM{
		vm:     goja.New(),
		loader: newLoader(baseDir),
	}
	result.vm.Set("os", nil)
	result.vm.Set("exec", nil)
	err := result.vm.Set("__loadModule", func(path string) (string, error) {
		code, err := result.loader.Load(path)
		if err != nil {
			return "", err
		}
		return code, nil
	})
	if err != nil {
		return nil, err
	}
	_, err = result.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	return result, nil
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := s.vm.RunString(string(data))
	if err != nil {
		return nil, err
	}
	return result, nil
}
