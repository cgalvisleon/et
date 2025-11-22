package vm

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/request"
	"github.com/dop251/goja"
)

/**
* Console
* @param vm *Vm
**/
func Console(vm *Vm) {
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
			logs.Error(fmt.Errorf(args))
		},
	})
}

/**
* Fetch
* @param vm *Vm
**/
func Fetch(vm *Vm) {
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 4 {
			panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		result, status := request.Fetch(method, url, headers, body)
		if status.Code != 200 {
			panic(vm.NewGoError(fmt.Errorf(status.Message)))
		}
		if !status.Ok {
			panic(vm.NewGoError(fmt.Errorf("error al hacer la peticion: %s", status.Message)))
		}

		return vm.ToValue(result)
	})
}

/**
* Event
* @param vm *Vm
**/
func Event(vm *Vm) {
	err := event.Load()
	if err != nil {
		return
	}

	vm.Set("event", map[string]interface{}{
		"publish": func(call goja.FunctionCall) {
			args := call.Arguments
			if len(args) != 2 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "channel, data")))
			}
			channel := args[0].String()
			data := args[1].Export().(map[string]interface{})
			event.Publish(channel, data)
		},
		"work": func(call goja.FunctionCall) {
			args := call.Arguments
			if len(args) != 2 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "channel, data")))
			}
			channel := args[0].String()
			data := args[1].Export().(map[string]interface{})
			event.Work(channel, data)
		},
	})
}

/**
* Cache
* @param vm *Vm
**/
func Cache(vm *Vm) {
	err := cache.Load()
	if err != nil {
		return
	}

	vm.Set("cache", map[string]interface{}{
		"set": func(call goja.FunctionCall) goja.Value {
			args := call.Arguments
			if len(args) != 3 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "key, value, expiration (minutes)")))
			}
			key := args[0].String()
			val := args[1].Export().(interface{})
			expMinutes := args[2].Export().(int64)
			expiration := time.Duration(expMinutes) * time.Minute
			result := cache.Set(key, val, expiration)
			return vm.ToValue(result)
		},
		"get": func(call goja.FunctionCall) goja.Value {
			args := call.Arguments
			if len(args) != 2 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "key, default")))
			}
			key := args[0].String()
			defVal := args[1].String()
			result, err := cache.Get(key, defVal)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(result)
		},
		"delete": func(call goja.FunctionCall) goja.Value {
			args := call.Arguments
			if len(args) != 1 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "key")))
			}
			key := args[0].String()
			result, err := cache.Delete(key)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(result)
		},
		"incr": func(call goja.FunctionCall) goja.Value {
			args := call.Arguments
			if len(args) != 2 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "key, expiration (seconds)")))
			}
			key := args[0].String()
			expSeconds := args[1].Export().(int64)
			result := cache.Incr(key, time.Duration(expSeconds)*time.Second)
			return vm.ToValue(result)
		},
		"decr": func(call goja.FunctionCall) goja.Value {
			args := call.Arguments
			if len(args) != 2 {
				panic(vm.NewGoError(fmt.Errorf(MSG_ARG_REQUIRED, "key")))
			}
			key := args[0].String()
			result := cache.Decr(key)
			return vm.ToValue(result)
		},
	})
}
