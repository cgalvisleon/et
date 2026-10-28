package jwf

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/validator"
	"github.com/dop251/goja"
)

/**
* GetArgs: The argument at idx of a goja call exported to Go, or defaultValue when idx is out of range.
* @param args []goja.Value, idx int, defaultValue any
* @return any
**/
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
* wrapperConsole: Adds the console binding (server log only, no emitter).
* @param bindings map[string]interface{}
* @return map[string]interface{}
**/
func wrapperConsole(bindings map[string]interface{}) map[string]interface{} {
	bindings["console"] = consoleOf("", nil)
	return bindings
}

/**
* wrapperBasic: Adds the basic bindings (ids, time, validator, evaluate*, field references, status constants…).
* @param bindings map[string]interface{}
* @return map[string]interface{}
**/
func wrapperBasic(bindings map[string]interface{}) map[string]interface{} {
	bindings["UUID"] = reg.UUID
	bindings["ULID"] = reg.ULID
	bindings["XID"] = reg.XID
	bindings["getUUID"] = reg.GetUUID
	bindings["getULID"] = reg.GetULID
	bindings["getXID"] = reg.GetXID
	bindings["timeNow"] = timezone.Now
	// sleep(ms): detiene el script esos milisegundos (el paso delay)
	bindings["sleep"] = func(ms int) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
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
	bindings["Json"] = Binder(func(vm *goja.Runtime) goja.Value { return et.ToJS(vm, et.Json{}) })
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
* fetchOptions: The optional arguments of fetch after method, url, headers and body: the timeout in seconds (0: no
* limit) and the body returned when the request fails (a text, or any value as its JSON)
* @param options []any
* @return time.Duration, []byte
**/
func fetchOptions(options []any) (time.Duration, []byte) {
	timeout := time.Duration(0)
	if len(options) > 0 {
		switch value := options[0].(type) {
		case int64:
			timeout = time.Duration(value) * time.Second
		case float64:
			timeout = time.Duration(value * float64(time.Second))
		}
	}

	var defaultValue []byte
	if len(options) > 1 && options[1] != nil {
		switch value := options[1].(type) {
		case string:
			defaultValue = []byte(value)
		default:
			defaultValue, _ = json.Marshal(value)
		}
	}
	return timeout, defaultValue
}

/**
* wrapperFetch: Wraps the fetch: fetch(method, url, headers, body, timeout?, defaultValue?) and
* fetchTls(method, url, headers, body, tlsConfig, timeout?, defaultValue?)
* @param bindings map[string]interface{}
**/
func wrapperFetch(bindings map[string]interface{}) map[string]interface{} {
	bindings["fetch"] = func(method, url string, headers, body et.Json, options ...any) *Fetch {
		timeout, defaultValue := fetchOptions(options)
		result, status := request.Fetch(method, url, headers, body, timeout, defaultValue)
		return &Fetch{Ok: status.Ok, Result: result, Status: status.Code, Message: status.Message}
	}
	bindings["fetchTls"] = func(method, url string, headers, body et.Json, tlsConfig *tls.Config, options ...any) *Fetch {
		timeout, defaultValue := fetchOptions(options)
		result, status := request.FetchWithTls(method, url, headers, body, tlsConfig, timeout, defaultValue)
		return &Fetch{Ok: status.Ok, Result: result, Status: status.Code, Message: status.Message}
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
* wrapperCache: Adds the cache binding (set, get, json, items, item, delete over the Redis cache).
* @param bindings map[string]interface{}
* @return map[string]interface{}
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
			return err == nil
		},
	}
	return bindings
}

/**
* wrapperEvent: Adds the event binding (publish and subscribe over NATS).
* @param bindings map[string]interface{}
* @return map[string]interface{}
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
* wrapperConsole: Sets the console binding of the instance; its lines also go to the OnLog hooks through emitLog.
**/
func (s *Instance) wrapperConsole() {
	s.SetBinding("console", consoleOf("", s.emitLog))
}

/**
* wrapperCtx: Sets the ctx binding of the instance over Instance.Ctx (created if nil).
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
* wrapperGoTo: Sets the goTo(idx) binding, which picks the output port the current step continues through.
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

/**
* wrapperAgain: Wraps the again
* @param instance *Instance
**/
func (s *Instance) wrapperAgain() {
	s.SetBinding("again", func() error {
		if s.Step == nil {
			return errors.New(MSG_STEP_NOT_FOUND)
		}
		s.setAgain()
		return nil
	})
}
