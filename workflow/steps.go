package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

type StepKind string

const (
	StepFn     StepKind = "fn"
	StepDefine StepKind = "define"
)

type Condition struct {
	Expression string `json:"expression"`
	YesTo      int    `json:"yes_to"`
	NoTo       int    `json:"no_to"`
}

type Step struct {
	Kind        StepKind    `json:"kind"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Stop        bool        `json:"stop"`
	Condition   *Condition  `json:"condition"`
	Definition  []byte      `json:"definition"`
	Steps       []*Step     `json:"steps"`
	fn          FnContext   `json:"-"`
	rollbacks   FnContext   `json:"-"`
	shot        *time.Timer `json:"-"`
}

/**
* newStep
* @param name, description, expression string, nextIndex int, fn FnContext, stop bool
* @return *Step
**/
func newStep(name, description string, fn FnContext, stop bool) (*Step, error) {
	result := &Step{
		Kind:        StepFn,
		Name:        name,
		Description: description,
		Stop:        stop,
		Definition:  []byte{},
		Condition:   &Condition{},
		Steps:       make([]*Step, 0),
		fn:          fn,
	}

	return result, nil
}

/**
* run
* @params flow *Instance, ctx et.Json
* @return et.Json, error
**/
func (s *Step) run(flow *Instance, ctx et.Json) (et.Json, error) {
	if s.Kind == StepDefine {
		return ctx, nil
	}

	if s.fn == nil {
		return ctx, fmt.Errorf(msg.MSG_STEP_FUNCTION_IS_NIL, s.Name, flow.Current)
	}

	flow.setStatus(Running)
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
* @param expression string, yesTo int, noTo int
* @return *Step, error
**/
func (s *Step) ifElse(expression string, yesTo int, noTo int) *Step {
	s.Condition = &Condition{
		Expression: expression,
		YesTo:      yesTo,
		NoTo:       noTo,
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
		return false, fmt.Errorf(MSG_INSTANCE_EVALUATE, s.Condition.Expression, err.Error())
	}

	instance.setStatus(Running)
	evalueExpression, err := govaluate.NewEvaluableExpression(s.Condition.Expression)
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
