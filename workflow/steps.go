package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/Knetic/govaluate"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/dop251/goja"
)

type TpStep string

const (
	TpFn         TpStep = "function"
	TpDefinition TpStep = "definition"
)

type FnContext func(flow *Instance, ctx et.Json) (et.Json, error)

type Step struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        TpStep    `json:"type"`
	Stop        bool      `json:"stop"`
	Expression  string    `json:"expression"`
	YesGoTo     int       `json:"yes_go_to"`
	NoGoTo      int       `json:"no_go_to"`
	Definition  string    `json:"definition"`
	fn          FnContext `json:"-"`
	rollbacks   FnContext `json:"-"`
}

/**
* newStepFn
* @param name, description, expression string, nextIndex int, fn FnContext, stop bool
* @return *Step
**/
func newStepFn(name, description string, fn FnContext, stop bool) (*Step, error) {
	result := &Step{
		Name:        name,
		Description: description,
		Type:        TpFn,
		Stop:        stop,
		fn:          fn,
	}

	return result, nil
}

/**
* newStepDefinition
* @param name, description, definition string, stop bool
* @return *Step
**/
func newStepDefinition(name, description string, definition string, stop bool) (*Step, error) {
	result := &Step{
		Name:        name,
		Description: description,
		Type:        TpDefinition,
		Stop:        stop,
		Definition:  definition,
	}
	result.fn = func(flow *Instance, ctx et.Json) (et.Json, error) {
		flow.vm.Set("instance", flow)
		flow.vm.Set("ctx", flow.Ctx)
		flow.vm.Set("ctxs", flow.Ctxs)
		flow.vm.Set("pinnedData", flow.PinnedData)
		if definition == "" {
			return et.Json{}, fmt.Errorf(MSG_INSTANCE_DEFINITION_EMPTY)
		}

		value, err := flow.vm.Run(definition)
		if err != nil {
			return et.Json{}, err
		}

		if flow.isDebug {
			logs.Debugf("stepDefinition:%s", value.String())
		}

		result := flow.vm.Get("result")
		if result == goja.Undefined() {
			return et.Json{}, nil
		}

		if result == goja.Null() {
			return et.Json{}, nil
		}

		if result == nil {
			return et.Json{}, nil
		}

		switch v := result.Export().(type) {
		case map[string]interface{}:
			return et.Json(v), nil
		case et.Json:
			return v, nil
		default:
			return et.Json{
				"result": v,
			}, nil
		}
	}

	return result, nil
}

/**
* run
* @params flow *Instance, ctx et.Json
* @return et.Json, error
**/
func (s *Step) run(flow *Instance, ctx et.Json) (et.Json, error) {
	flow.SetStatus(FlowStatusRunning)
	result, err := s.fn(flow, ctx)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Step) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Step) ToJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* ifElse
* @param expression string, yesGoTo int, noGoTo int
* @return *Step, error
**/
func (s *Step) ifElse(expression string, yesGoTo int, noGoTo int) *Step {
	s.YesGoTo = yesGoTo
	s.NoGoTo = noGoTo
	if expression != "" {
		s.Expression = expression
	}

	return s
}

/**
* evaluate
* @param ctx et.Json
* @return bool, error
**/
func (s *Step) evaluate(ctx et.Json, instance *Instance) (bool, error) {
	resultError := func(err error) (bool, error) {
		return false, fmt.Errorf(MSG_INSTANCE_EVALUATE, s.Expression, err.Error())
	}

	instance.SetStatus(FlowStatusRunning)
	evalueExpression, err := govaluate.NewEvaluableExpression(s.Expression)
	if err != nil {
		return resultError(err)
	}

	ok, err := evalueExpression.Evaluate(ctx)
	if err != nil {
		return resultError(err)
	}

	switch v := ok.(type) {
	case bool:
		return v, nil
	default:
		return resultError(fmt.Errorf("expression result is not a boolean"))
	}
}
