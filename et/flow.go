package et

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/logs"
)

type TypeRun int

const (
	Func TypeRun = iota
	Gorutine
	Gocontext
)

type GoContext func(ctx Json) (Item, error)

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

func Flow(fn interface{}, args []interface{}, tpRun TypeRun) *Fn {
	result := &Funcs{}

	return result.Add(fn, args, tpRun)
}

/**
* Add
* @param fn interface{}, args []interface{}, tpRun TypeRun
**/
func (s *Funcs) Add(fn interface{}, args []interface{}, tpRun TypeRun) *Fn {
	_fn := &Fn{
		owner: s,
		fn:    fn,
		args:  args,
		tpRun: tpRun,
	}

	*s = append(*s, _fn)

	return _fn
}

/**
* Run
* @return Item, error
**/
func (s *Funcs) Run() (Item, error) {
	var err error
	var ctx = Item{Result: Json{}}
	for i, f := range *s {
		funcName := getFunctionName(f.fn)
		logs.Logf("flow", "Execute func: %s step:%d - params:%s", funcName, i, fmt.Sprint(f.args))
		switch f.tpRun {
		case Gocontext:
			ctx, err = f.fn.(GoContext)(ctx.Result)
			if err != nil {
				result, rr := s.Rollback(i)
				if rr != nil {
					return Item{}, fmt.Errorf(`error step:%d - %s`, i, rr.Error())
				}
				return result, fmt.Errorf(`error step:%d - %s`, i, err.Error())
			}
			if !ctx.Ok {
				_, rr := s.Rollback(i)
				if rr != nil {
					return Item{}, fmt.Errorf(`error step:%d - %s`, i, rr.Error())
				}
			}
		default:
			fn := reflect.ValueOf(f.fn)
			if fn.Kind() != reflect.Func {
				return Item{}, fmt.Errorf(`error step:%d - %s is not a function`, i, funcName)
			}
			argsValues := make([]reflect.Value, len(f.args))
			for i, arg := range f.args {
				argsValues[i] = reflect.ValueOf(arg)
			}

			if f.tpRun == Gorutine {
				numArgs := fn.Type().NumIn()
				if len(argsValues) != numArgs {
					result, rr := s.Rollback(i)
					if rr != nil {
						return Item{}, fmt.Errorf(`error step:%d - %s`, i, rr.Error())
					}
					return result, fmt.Errorf(`error step:%d - %s`, i, "Call with too many input arguments")
				}
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

/**
* Rollback
* @param i int
* @return Item, error
**/
func (s *Funcs) Rollback(i int) (Item, error) {
	var err error
	var ctx = Item{Result: Json{}}
	for j := i; j >= 0; j-- {
		f := (*s)[j]
		if f.rollback.fn == nil {
			continue
		}

		funcName := getFunctionName(f.fn)
		logs.Logf("flow", "Rollback func: %s step:%d - params:%s", funcName, i, fmt.Sprint(f.args))
		switch f.tpRun {
		case Gocontext:
			ctx, err = f.rollback.fn.(GoContext)(ctx.Result)
			if err != nil {
				return Item{}, fmt.Errorf(`error step:%d - %s`, i, err.Error())
			}
			if !ctx.Ok {
				return ctx, nil
			}
		default:
			fn := reflect.ValueOf(f.fn)
			if fn.Kind() != reflect.Func {
				return Item{}, fmt.Errorf(`error step:%d - %s is not a function`, i, funcName)
			}
			argsValues := make([]reflect.Value, len(f.args))
			for i, arg := range f.args {
				argsValues[i] = reflect.ValueOf(arg)
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
