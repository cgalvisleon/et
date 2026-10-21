package jwf

import (
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/validator"
	"github.com/dop251/goja"
)

func GetArgs(args []goja.Value, idx int, defaultValue any) any {
	if idx < 0 || idx >= len(args) {
		return defaultValue
	}
	return args[idx].Export()
}

/**
* Bindings: The standard bindings of a script added to bindings (created if nil): console, the basic functions
* (UUID, timeNow, validator, evaluateObject, the status constants…), fetch, jrpc, cache and event; a key that bindings
* already has is replaced
* @param bindings map[string]interface{}
* @return map[string]interface{}
**/
func Bindings(bindings map[string]interface{}) map[string]interface{} {
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
	bindings["console"] = consoleOf("", nil)
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
	bindings["getUUID"] = reg.GetUUID
	bindings["getULID"] = reg.GetULID
	bindings["getXID"] = reg.GetXID
	bindings["timeNow"] = timezone.Now
	bindings["validator"] = validator.Wrapper()
	evaluateObject := et.EvaluateObjectWrapper()
	bindings["evaluateObject"] = func(obj et.Json, where any) (bool, error) {
		list := et.ToWhere(where)
		return evaluateObject(obj, list)
	}
	evaluateValue := et.EvaluateValueWrapper()
	bindings["evaluateValue"] = func(value any, where any, obj ...et.Json) (bool, error) {
		list := et.ToWhere(where)
		return evaluateValue(value, list, obj...)
	}
	for name, fn := range et.FieldWrapper() {
		bindings[name] = fn
	}
	bindings["isEditableStatus"] = IsEditableStatus
	bindings["isExcludedStatus"] = IsExcludedStatus
	bindings["statusList"] = statusList
	bindings["statusSystem"] = SYSTEM
	bindings["statusActive"] = ACTIVE
	bindings["statusArchived"] = ARCHIVED
	bindings["statusCanceled"] = CANCELED
	bindings["statusForDelete"] = FOR_DELETE
	bindings["statusInProcess"] = IN_PROCESS
	bindings["statusCreated"] = CREATED
	bindings["statusPending"] = PENDING
	bindings["statusRunning"] = RUNNING
	bindings["statusRollback"] = ROLLBACK
	bindings["statusDone"] = DONE
	bindings["statusApproved"] = APPROVED
	bindings["statusRejected"] = REJECTED
	bindings["statusFailed"] = FAILED
	bindings["statusCancel"] = CANCEL
	bindings["Json"] = et.Json{}
	bindings["replace"] = func(value string, obj et.Json) string {
		result := value
		for k, v := range obj {
			result = strings.ReplaceAll(result, fmt.Sprintf(`{{%v}}`, k), fmt.Sprintf(`%v`, v))
		}
		return result
	}
	bindings["hash"] = func(value string) string {
		return utility.Sha256(value)
	}
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
		headers, _ := et.JsonOf(args[2])
		body, _ := et.JsonOf(args[3])
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
		headers, _ := et.JsonOf(args[2])
		body, _ := et.JsonOf(args[3])
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
* aliasJson: The names of the methods of the params and ctx of before (str, int, json, arrayJson…), on the prototype of
* et.Json of the runtime, so the scripts written with them keep working: set merges an object (Set of et.Json sets a key),
* view keeps keys under another name and omit removes keys
* @param vm *goja.Runtime
**/
func aliasJson(vm *goja.Runtime) {
	proto := et.JsonPrototype(vm)
	names := map[string]string{
		"put": "Put", "get": "Get", "str": "Str", "int": "Int", "int64": "Int64", "num": "Num", "bool": "Bool", "time": "Time",
		"json": "Json", "array": "Array", "arrayStr": "ArrayStr", "arrayInt": "ArrayInt", "arrayInt64": "ArrayInt64", "arrayJson": "ArrayJson",
	}
	for alias, name := range names {
		proto.Set(alias, proto.Get(name))
	}

	this := func(call goja.FunctionCall) et.Json {
		data, ok := et.JsonOf(call.This)
		if !ok {
			panic(vm.NewTypeError("this is not an et.Json"))
		}
		return data
	}
	proto.Set("set", func(call goja.FunctionCall) goja.Value {
		values, _ := et.JsonOf(call.Argument(0))
		maps.Copy(this(call), values)
		return goja.Undefined()
	})
	proto.Set("view", func(call goja.FunctionCall) goja.Value {
		org := this(call)
		view, _ := et.JsonOf(call.Argument(0))
		result := et.Json{}
		for key, value := range view {
			as := fmt.Sprintf("%v", value)
			if as == "" {
				as = key
			}
			if _, ok := org[key]; ok {
				result[as] = org[key]
			}
		}
		if !result.IsEmpty() {
			org.Put(result)
		}
		return goja.Undefined()
	})
	proto.Set("omit", func(call goja.FunctionCall) goja.Value {
		org := this(call)
		for _, key := range call.Arguments {
			delete(org, key.String())
		}
		return goja.Undefined()
	})
}

/**
* jsonBinder: The binding of the json that data gives (created if missing), built per run with et.ToJS: an object over
* the same map, its keys as properties and the methods of et.Json
* @param data func() et.Json
* @return Binder
**/
func jsonBinder(data func() et.Json) Binder {
	return func(vm *goja.Runtime) goja.Value {
		return et.ToJS(vm, data())
	}
}

/**
* wrapperParams: Wraps the params: params.email reads a param and params.str("email") and the other methods still work
* @param s *Step
**/
func (s *Step) wrapperParams() {
	params := func() et.Json {
		if s.Params == nil {
			s.Params = et.Json{}
		}
		return s.Params
	}
	s.SetBinding("params", jsonBinder(params))
}

/**
* wrapperCtx: Wraps the ctx
* @param s *Step
**/
func (s *Step) wrapperCtx() {
	ctx := func() et.Json {
		if s.ctx == nil {
			s.ctx = et.Json{}
		}
		return s.ctx
	}
	s.SetBinding("ctx", jsonBinder(ctx))
}

/**
* wrapperCtx:
* @param s *Instance
**/
func (s *Instance) wrapperConsole() {
	s.SetBinding("console", consoleOf("", s.emitLog))
}

/**
* wrapperCtx: Wraps the ctx
* @param instance *Instance
**/
func (s *Instance) wrapperCtx() {
	ctx := func() et.Json {
		if s.Ctx == nil {
			s.Ctx = et.Json{}
		}
		return s.Ctx
	}
	s.SetBinding("ctx", jsonBinder(ctx))
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
