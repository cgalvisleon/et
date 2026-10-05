package jrex

import (
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/timezone"
	"github.com/dop251/goja"
)

func GetArgs(args []goja.Value, idx int, defaultValue any) any {
	if idx < 0 || idx >= len(args) {
		return defaultValue
	}
	return args[idx].Export()
}

/**
* wrap: Wraps the runtime
* @param vm *VM
**/
func wrapper(instance *Instance) {
	wrapperBasic(instance)
	wrapperVar(instance)
	wrapperConsole(instance)
	wrapperFetch(instance)
	wrapperJrpc(instance)
	wrapperCache(instance)
	wrapperEvent(instance)
}

/**
* wrapperBasic: Wraps the basic
* @param instance *Instance
**/
func wrapperBasic(instance *Instance) {
	instance.Set("UUID", reg.UUID)
	instance.Set("ULID", reg.ULID)
	instance.Set("XID", reg.XID)
	instance.Set("GetUUID", reg.GetUUID)
	instance.Set("GetULID", reg.GetULID)
	instance.Set("GetXID", reg.GetXID)
	instance.Set("timeNow", timezone.Now)
}

/**
* wrapperCtx: Wraps the ctx
* @param instance *Instance
**/
func wrapperVar(instance *Instance) {
	instance.Set("ctx", map[string]interface{}{
		"set": func(data et.Json) {
			maps.Copy(instance.Ctx, data)
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
* @param instance *Instance
**/
func wrapperConsole(instance *Instance) {
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

type Fetch struct {
	Ok      bool          `json:"ok"`
	Result  *request.Body `json:"result"`
	Status  int           `json:"status"`
	Message string        `json:"message"`
}

/**
* wrapperFetch: Wraps the fetch
* @param instance *Instance
**/
func wrapperFetch(instance *Instance) {
	instance.Set("fetch", func(call goja.FunctionCall) *Fetch {
		args := call.Arguments
		if len(args) != 4 {
			panic(instance.Error(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		duration, _ := GetArgs(args, 4, 0).(int64)
		timeout := time.Duration(duration) * time.Second
		defaultValue := GetArgs(args, 5, nil).([]byte)
		result, status := request.Fetch(method, url, headers, body, timeout, defaultValue)
		res := &Fetch{
			Ok:      status.Ok,
			Result:  result,
			Status:  status.Code,
			Message: status.Message,
		}
		return res
	})
	instance.Set("fetchTls", func(call goja.FunctionCall) *Fetch {
		args := call.Arguments
		if len(args) != 4 {
			panic(instance.Error(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		tlsConfig := GetArgs(args, 4, nil).(*tls.Config)
		duration, _ := GetArgs(args, 5, 0).(int64)
		timeout := time.Duration(duration) * time.Second
		defaultValue := GetArgs(args, 6, nil).([]byte)
		result, status := request.FetchWithTls(method, url, headers, body, tlsConfig, timeout, defaultValue)
		res := &Fetch{
			Ok:      status.Ok,
			Result:  result,
			Status:  status.Code,
			Message: status.Message,
		}
		return res
	})
}

/**
* wrapperJrpc: Wraps the jrpc
* @param instance *Instance
**/
func wrapperJrpc(instance *Instance) {
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
* @param instance *Instance
**/
func wrapperCache(instance *Instance) {
	instance.Set("cache", map[string]interface{}{
		"set": func(key string, value interface{}, expiration time.Duration) interface{} {
			return cache.Set(key, value, expiration)
		},
		"get": func(key string, defaultValue string) string {
			result, _, err := cache.Get(key, defaultValue)
			if err != nil {
				return defaultValue
			}
			return result
		},
		"json": func(key string) et.Json {
			result, _, err := cache.GetJson(key)
			if err != nil {
				return et.Json{}
			}
			return result
		},
		"items": func(key string) et.Items {
			result, _, err := cache.GetItems(key)
			if err != nil {
				return et.Items{}
			}
			return result
		},
		"item": func(key string) et.Item {
			result, _, err := cache.GetItem(key)
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

/**
* wrapperEvent: Wraps the event
* @param instance *Instance
**/
func wrapperEvent(instance *Instance) {
	instance.Set("event", map[string]interface{}{
		"publish": func(channel string, data et.Json) {
			event.Publish(channel, data)
		},
	})
}
