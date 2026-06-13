package jrex

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
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
	wrapperJrpc(instance)
	wrapperCache(instance)
	wrapperEvent(instance)
	for _, module := range instance.Modules {
		wrapperModules(module)
	}
}

/**
* wrapperRunTime: Wraps the runtime
* @param vm *VM
**/
func wrapperRunTime(instance *Jrex) {
	instance.Set("__load", func(module string) string {
		code, err := instance.store.GetCode(module)
		if err != nil {
			panic(instance.Error(err))
		}
		return code
	})
}

/**
* wrapperRunTime: Wraps the runtime
* @param vm *VM
**/
func wrapperModules(module *Module) {
	module.Set("os", nil)
	module.Set("exec", nil)
	module.Set("version", func(value string) string {
		part, ok := ToPart(value)
		if !ok {
			panic(module.Error(fmt.Errorf("invalid part: %s", value)))
		}
		module.SetVersion(part)
		return module.Version
	})
	module.Set("description", func(value string) string {
		module.SetDescription(value)
		return module.Description
	})
	module.Set("author", func(value string) string {
		module.SetAuthor(value)
		return module.Author
	})
	module.Set("license", func(value string) string {
		module.SetLicense(value)
		return module.License
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

/**
* wrapperJrpc: Wraps the jrpc
* @param vm *VM
**/
func wrapperJrpc(instance *Jrex) {
	instance.Set("jrpc", map[string]interface{}{
		"call": func(method string, args any) (any, error) {
			return jrpc.Call(method, args)
		},
		"callJson": func(method string, args et.Json) (et.Json, error) {
			return jrpc.CallJson(method, args)
		},
		"callItems": func(method string, args et.Json) (et.Items, error) {
			return jrpc.CallItems(method, args)
		},
		"callItem": func(method string, args et.Json) (et.Item, error) {
			return jrpc.CallItem(method, args)
		},
	})
}

/**
* wrapperCache: Wraps the cache
* @param vm *VM
**/
func wrapperCache(instance *Jrex) {
	instance.Set("cache", map[string]interface{}{
		"set": func(key string, value interface{}, expiration time.Duration) interface{} {
			return cache.Set(key, value, expiration)
		},
		"get": func(key string, defaultValue string) string {
			result, err := cache.Get(key, defaultValue)
			if err != nil {
				return defaultValue
			}
			return result
		},
		"json": func(key string) et.Json {
			result, err := cache.GetJson(key)
			if err != nil {
				return et.Json{}
			}
			return result
		},
		"items": func(key string) et.Items {
			result, err := cache.GetItems(key)
			if err != nil {
				return et.Items{}
			}
			return result
		},
		"item": func(key string) et.Item {
			result, err := cache.GetItem(key)
			if err != nil {
				return et.Item{}
			}
			return result
		},
		"delete": func(key string) bool {
			_, err := cache.Delete(key)
			if err != nil {
				return false
			}
			return true
		},
	})
}

func wrapperEvent(instance *Jrex) {
	instance.Set("event", map[string]interface{}{
		"publish": func(channel string, data et.Json) {
			event.Publish(channel, data)
		},
	})
}
