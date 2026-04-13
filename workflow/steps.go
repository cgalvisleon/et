package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
)

type StepKind string

const (
	StepNormal StepKind = "normal"
	StepWait   StepKind = "wait"
)

type Step struct {
	Kind        StepKind      `json:"kind"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Stop        bool          `json:"stop"`
	Expression  string        `json:"expression"`
	YesGoTo     int           `json:"yes_go_to"`
	NoGoTo      int           `json:"no_go_to"`
	Spec        string        `json:"spec"`
	fn          FnContext     `json:"-"`
	rollbacks   FnContext     `json:"-"`
	duration    time.Duration `json:"-"`
	shot        *time.Timer   `json:"-"`
}

/**
* newStep
* @param name, description, expression string, nextIndex int, fn FnContext, stop bool
* @return *Step
**/
func newStep(name, description string, fn FnContext, stop bool) (*Step, error) {
	result := &Step{
		Kind:        StepNormal,
		Name:        name,
		Description: description,
		Stop:        stop,
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
	if s.fn == nil {
		return ctx, fmt.Errorf(msg.MSG_STEP_FUNCTION_IS_NIL, s.Name, flow.Current)
	}

	if s.Kind == StepWait {
		now := timezone.Now()
		shotTime, err := timezone.Parse("2006-01-02T15:04:05", s.Spec)
		if err != nil {
			return et.Json{}, err
		}
		if shotTime.After(now) {
			duration := shotTime.Sub(now)
			s.duration = duration
			s.shot = time.AfterFunc(duration, func() {
				s.fn(flow, ctx)
			})
		}

		return ctx, nil
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

	instance.setStatus(Running)
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
