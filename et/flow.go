package et

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/logs"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type TypeRun int

const (
	Func TypeRun = iota
	Gorutine
	Gocontext
)

type GoContext func(ctx Item) (Item, error)

type Fn struct {
	owner    *Funcs
	fn       interface{}
	args     []interface{}
	rollback struct {
		fn   interface{}
		args []interface{}
	}
	tpRun TypeRun
}

/**
* Rollback
* @param fn interface{}, args []interface{}
**/
func (s *Fn) Rollback(fn interface{}, args []interface{}) *Funcs {
	s.rollback = struct {
		fn   interface{}
		args []interface{}
	}{fn: fn, args: args}

	return s.owner
}

/**
* Add
* @param fn interface{}, args []interface{}, tpRun TypeRun
**/
func (s *Fn) Add(fn interface{}, args []interface{}, tpRun TypeRun) *Funcs {
	s.owner.Add(fn, args, tpRun)

	return s.owner
}

/**
* Run
* @return Item, error
**/
func (s *Fn) Run() (Item, error) {
	return s.owner.Run()
}

/**
* getFunctionName
* @param i interface{}
* @return string
**/
func getFunctionName(i interface{}) string {
	result := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	lst := strings.Split(result, "/")

	return lst[len(lst)-1]
}

type Funcs []*Fn

/**
* Flow
* @param fn interface{}, args []interface{}, tpRun ...TypeRun
* @return *Fn
**/
func Flow(fn interface{}, args []interface{}, tpRun ...TypeRun) *Fn {
	result := &Funcs{}
	tp := Func
	if len(tpRun) > 0 {
		tp = tpRun[0]
	}

	return result.Add(fn, args, tp)
}

/**
* Add
* @param fn interface{}, args []interface{}, tpRun TypeRun
**/
func (s *Funcs) Add(fn interface{}, args []interface{}, tpRun ...TypeRun) *Fn {
	tp := Func
	if len(tpRun) > 0 {
		tp = tpRun[0]
	}

	_fn := &Fn{
		owner: s,
		fn:    fn,
		args:  args,
		tpRun: tp,
	}

	*s = append(*s, _fn)

	return _fn
}

/**
* Run
* @return Item, error
**/
func (s *Funcs) Run() (Item, error) {
	var ctx = Item{Result: Json{}}
	for i, f := range *s {
		funcName := getFunctionName(f.fn)
		fn := reflect.ValueOf(f.fn)
		if fn.Kind() != reflect.Func {
			return Item{}, fmt.Errorf(`error step:%d - %s is not a function`, i, funcName)
		}
		argsValues := make([]reflect.Value, len(f.args))
		for j, arg := range f.args {
			argsValues[j] = reflect.ValueOf(arg)
		}

		logs.Logf("workflow", "Execute func:%s step:%d", funcName, i)
		switch f.tpRun {
		case Gocontext:
			if i == 0 {
				if val, ok := argsValues[0].Interface().(Json); ok {
					ctx = Item{Ok: true, Result: val}
				}
			}

			result, err := f.fn.(GoContext)(ctx)
			if err != nil {
				result, _ := s.Rollback(i, ctx)
				return result, fmt.Errorf(`error func:%s, step:%d - %s`, funcName, i, err.Error())
			}
			ctx = result
		default:
			if f.tpRun == Gorutine {
				numArgs := fn.Type().NumIn()
				if len(argsValues) != numArgs {
					result, _ := s.Rollback(i, ctx)
					return result, fmt.Errorf(`error func:%s, step:%d - %s`, funcName, i, "Call with too many input arguments")
				}
				go func() {
					fn.Call(argsValues)
				}()
				continue
			} else {
				result := fn.Call(argsValues)
				if len(result) == 0 {
					continue
				}
				for _, r := range result {
					if r.Type().Implements(errorInterface) {
						err, ok := r.Interface().(error)
						if ok && err != nil {
							result, _ := s.Rollback(i, ctx)
							return result, fmt.Errorf(`error func:%s, step:%d - %s`, funcName, i, err.Error())
						}
					}
				}
			}
		}
	}

	return ctx, nil
}

/**
* Do
* @return Item, error
**/
func (s *Funcs) Do() (Item, error) {
	return s.Run()
}

/**
* Rollback
* @param i int
* @return Item, error
**/
func (s *Funcs) Rollback(i int, ctx Item) (Item, error) {
	for j := i; j >= 0; j-- {
		f := (*s)[j]
		if f.rollback.fn == nil {
			continue
		}

		funcName := getFunctionName(f.fn)
		fn := reflect.ValueOf(f.fn)
		if fn.Kind() != reflect.Func {
			return Item{}, fmt.Errorf(`error step:%d - %s is not a function`, j, funcName)
		}
		argsValues := make([]reflect.Value, len(f.args))
		for j, arg := range f.args {
			argsValues[j] = reflect.ValueOf(arg)
		}

		logs.Logf("workflow", "Rollback func: %s step:%d - params:%s", funcName, j, fmt.Sprint(f.args))
		switch f.tpRun {
		case Gocontext:
			result, err := f.rollback.fn.(GoContext)(ctx)
			if err == nil {
				ctx = result
			}
		default:
			fn := reflect.ValueOf(f.fn)
			if fn.Kind() != reflect.Func {
				return Item{}, fmt.Errorf(`error step:%d - %s is not a function`, j, funcName)
			}

			if f.tpRun == Gorutine {
				go func() {
					fn.Call(argsValues)
				}()
				continue
			} else {
				fn.Call(argsValues)
			}
		}
	}

	return ctx, nil
}
