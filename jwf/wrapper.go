package jwf

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
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
	bindings["evaluateObject"] = func(obj et.Json, where map[string]any) (bool, error) {
		list := et.ToWhere(where)
		return evaluateObject(obj, list)
	}
	evaluateValue := et.EvaluateValueWrapper()
	bindings["evaluateValue"] = func(value any, where map[string]any, obj ...et.Json) (bool, error) {
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

// jsonObject: an et.Json in JavaScript (params of a step, ctx of an instance): its keys as properties (params.email,
// JSON.stringify(ctx)) plus, by their name, the methods of jsonMethods (str, int, set…); a method hides a key with the
// same name (read it with get). data gives the json it reads and writes (created if missing)
type jsonObject struct {
	data    func() et.Json
	vm      *goja.Runtime
	methods map[string]any
}

func (o *jsonObject) Get(key string) goja.Value {
	if method, ok := o.methods[key]; ok {
		return o.vm.ToValue(method)
	}
	value, ok := o.data()[key]
	if !ok {
		return nil
	}
	return o.vm.ToValue(plainOf(value))
}

/**
* plainOf: The value with its et.Json (also nested or in lists) as map[string]any: goja wraps the named type et.Json
* as a Go object with its methods, so its keys would not be readable from JavaScript (ctx.user.name)
* @param value any
* @return any
**/
func plainOf(value any) any {
	switch v := value.(type) {
	case et.Json:
		return plainOf(map[string]any(v))
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, item := range v {
			result[key] = plainOf(item)
		}
		return result
	case []et.Json:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = plainOf(item)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = plainOf(item)
		}
		return result
	}
	return value
}

func (o *jsonObject) Set(key string, val goja.Value) bool {
	o.data()[key] = val.Export()
	return true
}

func (o *jsonObject) Has(key string) bool {
	_, ok := o.data()[key]
	return ok
}

func (o *jsonObject) Delete(key string) bool {
	delete(o.data(), key)
	return true
}

func (o *jsonObject) Keys() []string {
	return slices.Sorted(maps.Keys(o.data()))
}

// String and MarshalJSON: the object passed to Go (console.debug(params), a log, a json) shows its values
func (o *jsonObject) String() string {
	return o.data().ToString()
}

func (o *jsonObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.data())
}

/**
* jsonMethods: The methods of a jsonObject over the json that data gives: set (merge) and the typed reads of et.Json
* @param data func() et.Json
* @return map[string]any
**/
func jsonMethods(data func() et.Json) map[string]any {
	return map[string]any{
		"set": func(values et.Json) {
			maps.Copy(data(), values)
		},
		"put": func(values et.Json) {
			data().Put(values)
		},
		"view": func(view et.Json) {
			result := et.Json{}
			org := data()
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
				data().Put(result)
			}
		},
		"omit": func(keys ...string) {
			for _, key := range keys {
				delete(data(), key)
			}
		},
		"get": func(keys ...string) any {
			return data().Get(keys...)
		},
		"str": func(keys ...string) string {
			return data().Str(keys...)
		},
		"int": func(keys ...string) int {
			return data().Int(keys...)
		},
		"int64": func(keys ...string) int64 {
			return data().Int64(keys...)
		},
		"num": func(keys ...string) float64 {
			return data().Num(keys...)
		},
		"bool": func(keys ...string) bool {
			return data().Bool(keys...)
		},
		"time": func(keys ...string) time.Time {
			return data().Time(keys...)
		},
		"json": func(key string) et.Json {
			return data().Json(key)
		},
		"array": func(key string) []any {
			return data().Array(key)
		},
		"arrayStr": func(key string) []string {
			return data().ArrayStr(key)
		},
		"arrayInt": func(key string) []int {
			return data().ArrayInt(key)
		},
		"arrayInt64": func(key string) []int64 {
			return data().ArrayInt64(key)
		},
		"arrayJson": func(key string) []et.Json {
			return data().ArrayJson(key)
		},
	}
}

/**
* jsonBinder: The binding of a jsonObject over the json that data gives, built per run with its runtime
* @param data func() et.Json
* @return Binder
**/
func jsonBinder(data func() et.Json) Binder {
	methods := jsonMethods(data)
	return func(vm *goja.Runtime) goja.Value {
		object := &jsonObject{data: data, vm: vm, methods: methods}
		return vm.NewDynamicObject(object)
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
