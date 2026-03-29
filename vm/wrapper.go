package vm

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/dop251/goja"
)

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
