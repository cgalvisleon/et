package jwf

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
func wrapper(bindings map[string]interface{}) map[string]interface{} {
	if bindings == nil {
		bindings = make(map[string]interface{})
	}
	bindings = wrapperConsole(bindings)
	bindings = wrapperBasic(bindings)
	bindings = wrapperFetch(bindings)
	bindings = wrapperJrpc(bindings)
	bindings = wrapperCache(bindings)
	bindings = wrapperEvent(bindings)
	return bindings
}

/**
* wrapperConsole:
* @param instance *Instance
**/
func wrapperConsole(bindings map[string]interface{}) map[string]interface{} {
	bindings["console"] = map[string]interface{}{
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
	}
	return bindings
}

/**
* wrapperBasic: Wraps the basic
* @param instance *Instance
**/
func wrapperBasic(bindings map[string]interface{}) map[string]interface{} {
	bindings["UUID"] = reg.UUID
	bindings["ULID"] = reg.ULID
	bindings["XID"] = reg.XID
	bindings["GetUUID"] = reg.GetUUID
	bindings["GetULID"] = reg.GetULID
	bindings["GetXID"] = reg.GetXID
	bindings["timeNow"] = timezone.Now
	return bindings
}

type Fetch struct {
	Ok      bool          `json:"ok"`
	Result  *request.Body `json:"result"`
	Status  int           `json:"status"`
	Message string        `json:"message"`
}

/**
* wrapperFetch: Wraps the fetch
* @param bindings map[string]interface{}
**/
func wrapperFetch(bindings map[string]interface{}) map[string]interface{} {
	bindings["fetch"] = func(call goja.FunctionCall) *Fetch {
		args := call.Arguments
		if len(args) != 4 {
			panic(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body"))
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
	}
	bindings["fetchTls"] = func(call goja.FunctionCall) *Fetch {
		args := call.Arguments
		if len(args) != 4 {
			panic(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body"))
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
	}
	return bindings
}

/**
* wrapperJrpc: Wraps the jrpc
* @param bindings map[string]interface{}
**/
func wrapperJrpc(bindings map[string]interface{}) map[string]interface{} {
	bindings["jrpc"] = map[string]interface{}{
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
	}
	return bindings
}

/**
* wrapperCache: Wraps the cache
* @param instance *Instance
**/
func wrapperCache(bindings map[string]interface{}) map[string]interface{} {
	bindings["cache"] = map[string]interface{}{
		"set": func(key string, value interface{}, expiration time.Duration) interface{} {
			return cache.Set(key, value, expiration)
		},
		"get": func(key string, defaultValue string) string {
			result, exists, err := cache.Get(key, defaultValue)
			if err != nil {
				return defaultValue
			}
			if !exists {
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
	}
	return bindings
}

/**
* wrapperEvent: Wraps the event
* @param instance *Instance
**/
func wrapperEvent(bindings map[string]interface{}) map[string]interface{} {
	bindings["event"] = map[string]interface{}{
		"publish": func(channel string, data et.Json) {
			event.Publish(channel, data)
		},
		"subscribe": func(channel string, fn func(event.Message)) {
			event.Subscribe(channel, fn)
		},
	}
	return bindings
}

/**
* wrapperCtx:
* @param s *Instance
**/
func (s *Instance) wrapperConsole() {
	s.SetBinding("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			kind := "LOG"
			_args := make([]interface{}, 0)
			_args = append(_args, fmt.Sprintf(`host:%s - instance:%s`, s.hostname, s.ID))
			for _, arg := range args {
				_args = append(_args, arg)
			}
			logs.Log(kind, _args...)
		},
		"debug": func(args ...interface{}) {
			_args := make([]interface{}, 0)
			_args = append(_args, fmt.Sprintf(`host:%s - instance:%s`, s.hostname, s.ID))
			for _, arg := range args {
				_args = append(_args, arg)
			}
			logs.Debug(_args...)
		},
		"info": func(args ...interface{}) {
			_args := make([]interface{}, 0)
			_args = append(_args, fmt.Sprintf(`host:%s - instance:%s`, s.hostname, s.ID))
			for _, arg := range args {
				_args = append(_args, arg)
			}
			logs.Info(_args...)
		},
		"error": func(args string) {
			err := fmt.Errorf(`host:%s - instance:%s - %s`, s.hostname, s.ID, args)
			logs.Error(err)
		},
	})
}

/**
* wrapperCtx: Wraps the ctx
* @param instance *Instance
**/
func (s *Instance) wrapperCtx() {
	s.SetBinding("ctx", map[string]interface{}{
		"set": func(data et.Json) {
			maps.Copy(s.Ctx, data)
		},
		"get": func(keys ...string) interface{} {
			return s.Ctx.Get(keys...)
		},
		"str": func(keys ...string) string {
			return s.Ctx.Str(keys...)
		},
		"int": func(keys ...string) int {
			return s.Ctx.Int(keys...)
		},
		"int64": func(keys ...string) int64 {
			return s.Ctx.Int64(keys...)
		},
		"num": func(keys ...string) float64 {
			return s.Ctx.Num(keys...)
		},
		"bool": func(keys ...string) bool {
			return s.Ctx.Bool(keys...)
		},
		"time": func(keys ...string) time.Time {
			return s.Ctx.Time(keys...)
		},
		"json": func(key string) et.Json {
			return s.Ctx.Json(key)
		},
		"array": func(key string) []interface{} {
			return s.Ctx.Array(key)
		},
		"arrayStr": func(key string) []string {
			return s.Ctx.ArrayStr(key)
		},
		"arrayInt": func(key string) []int {
			return s.Ctx.ArrayInt(key)
		},
		"arrayInt64": func(key string) []int64 {
			return s.Ctx.ArrayInt64(key)
		},
		"arrayJson": func(key string) []et.Json {
			return s.Ctx.ArrayJson(key)
		},
	})
}

/**
* wrapperParams: Wraps the params
* @param instance *Instance
**/
func (s *Instance) wrapperParams() {
	s.SetBinding("params", map[string]interface{}{
		"set": func(data et.Json) {
			maps.Copy(s.Params, data)
		},
		"get": func(keys ...string) interface{} {
			return s.Params.Get(keys...)
		},
		"str": func(keys ...string) string {
			return s.Params.Str(keys...)
		},
		"int": func(keys ...string) int {
			return s.Params.Int(keys...)
		},
		"int64": func(keys ...string) int64 {
			return s.Params.Int64(keys...)
		},
		"num": func(keys ...string) float64 {
			return s.Params.Num(keys...)
		},
		"bool": func(keys ...string) bool {
			return s.Params.Bool(keys...)
		},
		"time": func(keys ...string) time.Time {
			return s.Params.Time(keys...)
		},
		"json": func(key string) et.Json {
			return s.Params.Json(key)
		},
		"array": func(key string) []interface{} {
			return s.Params.Array(key)
		},
		"arrayStr": func(key string) []string {
			return s.Params.ArrayStr(key)
		},
		"arrayInt": func(key string) []int {
			return s.Params.ArrayInt(key)
		},
		"arrayInt64": func(key string) []int64 {
			return s.Params.ArrayInt64(key)
		},
		"arrayJson": func(key string) []et.Json {
			return s.Params.ArrayJson(key)
		},
	})
}

/**
* wrapper
* @param step *Step
**/
func (s *Instance) wrapperGoTo() {
	s.SetBinding("goTo", func(idx int) error {
		if s.Step == nil {
			return errors.New(MSG_STEP_NOT_FOUND)
		}
		if idx < 0 || idx >= s.Step.Outputs {
			return errors.New(MSG_INVALID_INDEX)
		}
		s.setGoto(idx)
		return nil
	})
}
