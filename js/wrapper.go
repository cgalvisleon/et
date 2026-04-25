package js

import (
	"errors"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/dop251/goja"
)

/**
* wrap: Wraps the runtime
* @param vm *VM
**/
func wrap(vm *VM) {
	wrapperRunTime(vm)
	wrapperCtx(vm)
	wrapperConsole(vm)
	wrapperFetch(vm)
}

/**
* wrapperRunTime: Wraps the runtime
* @param vm *VM
**/
func wrapperRunTime(vm *VM) {
	vm.Set("os", nil)
	vm.Set("exec", nil)
	vm.Set("__rootDir", vm.Loader.BaseDir)
	vm.Set("__resolve", func(module, currentDir string) string {
		if vm.mode == Production {
			return fmt.Sprintf("pkg:%s:%s:%s", vm.Name, vm.Version, module)
		}

		p, err := vm.Loader.Resolve(module, currentDir)
		if err != nil {
			panic(vm.Error(err))
		}

		vm.SetModel(module, p)
		return p
	})
	vm.Set("__load", func(path string) string {
		if vm.mode == Production {
			var scr Module
			exists, err := vm.get(path, &scr)
			if err != nil {
				panic(vm.Error(err))
			}

			if !exists {
				panic(vm.Error(fmt.Errorf("script not found: %s", path)))
			}

			return scr.Scripts
		}

		inf := file.ExistPath(path)
		if inf.IsDir {
			return ""
		}
		data, err := os.ReadFile(path)
		if err != nil {
			panic(vm.Error(err))
		}
		return string(data)
	})
	vm.Set("version", func(value string) string {
		err := vm.SetVersion(value)
		if err != nil {
			panic(vm.Error(err))
		}
		return value
	})
	vm.Set("description", func(value string) string {
		err := vm.SetDescription(value)
		if err != nil {
			panic(vm.Error(err))
		}
		return value
	})
	vm.Set("author", func(value string) string {
		err := vm.SetAuthor(value)
		if err != nil {
			panic(vm.Error(err))
		}
		return value
	})
	vm.Set("license", func(value string) string {
		err := vm.SetLicense(value)
		if err != nil {
			panic(vm.Error(err))
		}
		return value
	})
}

/**
* wrapperCtx: Wraps the ctx
* @param vm *VM
**/
func wrapperCtx(vm *VM) {
	vm.Set("ctx", map[string]interface{}{
		"set": func(key string, value interface{}) interface{} {
			vm.Ctx.Set(key, value)
			return vm.Ctx
		},
		"get": func(key string) interface{} {
			return vm.Ctx.Get(key)
		},
		"delete": func(keys []string) interface{} {
			vm.Ctx.Delete(keys)
			return vm.Ctx
		},
		"exist": func(key string) bool {
			return vm.Ctx.Exist(key)
		},
	})
}

/**
* wrapperConsole: Wraps the console
* @param vm *VM
**/
func wrapperConsole(vm *VM) {
	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			kind := "Log"
			logs.Log(kind, args...)
		},
		"debug": func(args ...interface{}) {
			logs.Debug(args...)
		},
		"info": func(args ...interface{}) {
			logs.Info(args...)
		},
		"error": func(args string) {
			logs.Error(errors.New(args))
		},
	})
}

/**
* wrapperFetch: Wraps the fetch
* @param vm *VM
**/
func wrapperFetch(vm *VM) {
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 4 {
			panic(vm.Error(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		result, status := request.Fetch(method, url, headers, body)
		if status.Code != 200 {
			panic(vm.Error(errors.New(status.Message)))
		}
		if !status.Ok {
			panic(vm.Error(fmt.Errorf("error al hacer la peticion: %s", status.Message)))
		}
		return vm.Value(result)
	})
}
