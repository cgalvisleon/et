package jrex

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cgalvisleon/et/et"
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
func wrap(instance *Jrex) {
	wrapperRunTime(instance)
	wrapperCtx(instance)
	wrapperConsole(instance)
	wrapperFetch(instance)
}

/**
* wrapperRunTime: Wraps the runtime
* @param vm *VM
**/
func wrapperRunTime(instance *Jrex) {
	instance.Set("os", nil)
	instance.Set("exec", nil)
	instance.Set("__rootDir", instance.Loader.BaseDir)
	instance.Set("__resolve", func(module, currentDir string) string {
		if instance.mode == Production {
			return fmt.Sprintf("pkg:%s:%s:%s", instance.Name, module, instance.Version)
		}

		p, err := instance.Loader.Resolve(module, currentDir)
		if err != nil {
			panic(instance.Error(err))
		}

		instance.SetModule(module, p)
		return p
	})
	instance.Set("__load", func(path string) string {
		if instance.mode == Production {
			var scr Module
			exists, err := instance.get(path, &scr)
			if err != nil {
				panic(instance.Error(err))
			}

			if !exists {
				panic(instance.Error(fmt.Errorf("script not found: %s", path)))
			}

			return scr.Scripts
		}

		inf := file.ExistPath(path)
		if inf.IsDir {
			return ""
		}
		data, err := os.ReadFile(path)
		if err != nil {
			panic(instance.Error(err))
		}
		return string(data)
	})
	instance.Set("version", func(value string) string {
		err := instance.SetVersion(value)
		if err != nil {
			panic(instance.Error(err))
		}
		return value
	})
	instance.Set("description", func(value string) string {
		err := instance.SetDescription(value)
		if err != nil {
			panic(instance.Error(err))
		}
		return value
	})
	instance.Set("author", func(value string) string {
		err := instance.SetAuthor(value)
		if err != nil {
			panic(instance.Error(err))
		}
		return value
	})
	instance.Set("license", func(value string) string {
		err := instance.SetLicense(value)
		if err != nil {
			panic(instance.Error(err))
		}
		return value
	})
}

/**
* wrapperCtx: Wraps the ctx
* @param vm *VM
**/
func wrapperCtx(instance *Jrex) {
	instance.Set("ctx", map[string]interface{}{
		"set": func(key string, value interface{}) {
			instance.Ctx.Set(key, value)
		},
		"get": func(keys ...string) interface{} {
			return instance.Ctx.Get(keys...)
		},
		"str": func(keys ...string) string {
			return instance.Ctx.Str(keys...)
		},
		"int": func(keys ...string) int {
			return instance.Ctx.Int(keys...)
		},
		"int64": func(keys ...string) int64 {
			return instance.Ctx.Int64(keys...)
		},
		"num": func(keys ...string) float64 {
			return instance.Ctx.Num(keys...)
		},
		"bool": func(keys ...string) bool {
			return instance.Ctx.Bool(keys...)
		},
		"time": func(keys ...string) time.Time {
			return instance.Ctx.Time(keys...)
		},
		"json": func(key string) et.Json {
			return instance.Ctx.Json(key)
		},
		"array": func(key string) []interface{} {
			return instance.Ctx.Array(key)
		},
		"arrayStr": func(key string) []string {
			return instance.Ctx.ArrayStr(key)
		},
		"arrayInt": func(key string) []int {
			return instance.Ctx.ArrayInt(key)
		},
		"arrayInt64": func(key string) []int64 {
			return instance.Ctx.ArrayInt64(key)
		},
		"arrayJson": func(key string) []et.Json {
			return instance.Ctx.ArrayJson(key)
		},
	})
}

/**
* wrapperConsole: Wraps the console
* @param vm *VM
**/
func wrapperConsole(instance *Jrex) {
	instance.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			kind := "LOG"
			if instance.mode == Production {
				logs.Log(kind, args...)
			} else {
				instance.notify(kind, fmt.Sprintf("%v", args...))
			}
		},
		"debug": func(args ...interface{}) {
			kind := "DEBUG"
			if instance.mode == Production {
				logs.Debug(args...)
			} else {
				instance.notify(kind, fmt.Sprintf("%v", args...))
			}
		},
		"info": func(args ...interface{}) {
			kind := "INFO"
			if instance.mode == Production {
				logs.Info(args...)
			} else {
				instance.notify(kind, fmt.Sprintf("%v", args...))
			}
		},
		"error": func(args string) {
			kind := "ERROR"
			if instance.mode == Production {
				logs.Error(errors.New(args))
			} else {
				instance.notify(kind, args)
			}
		},
	})
}

/**
* wrapperFetch: Wraps the fetch
* @param vm *VM
**/
func wrapperFetch(instance *Jrex) {
	instance.Set("fetch", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 4 {
			panic(instance.Error(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		result, status := request.Fetch(method, url, headers, body)
		if status.Code != 200 {
			panic(instance.Error(errors.New(status.Message)))
		}
		if !status.Ok {
			panic(instance.Error(fmt.Errorf("error al hacer la peticion: %s", status.Message)))
		}
		return instance.Value(result)
	})
}
