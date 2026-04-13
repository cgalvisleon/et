package workflow

import (
	"encoding/json"
	"os"
	"slices"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

type TpConsistency string

const (
	TpConsistencyStrong   TpConsistency = "strong"
	TpConsistencyEventual TpConsistency = "eventual"
)

var workerHost string

func init() {
	workerHost, _ = os.Hostname()
}

type CheckList struct {
	Tag         string  `json:"tag"`
	Description string  `json:"description"`
	Ok          bool    `json:"ok"`
	Data        et.Json `json:"data"`
}

type FnContext func(flow *Instance, ctx et.Json) (et.Json, error)

type Flow struct {
	Tag           string          `json:"tag"`
	Version       string          `json:"version"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	TotalAttempts int             `json:"total_attempts"`
	TimeAttempts  time.Duration   `json:"time_attempts"`
	Steps         []*Step         `json:"steps"`
	TpConsistency TpConsistency   `json:"tp_consistency"`
	CheckList     []*CheckList    `json:"check_list"`
	Team          string          `json:"team"`
	Level         string          `json:"level"`
	CreatedBy     string          `json:"created_by"`
	onDone        func(*Instance) `json:"-"`
	isDebug       bool            `json:"-"`
}

/**
* newFlow
* @param workFlows *WorkFlows, tag, version, name, description string, fn FnContext, totalAttempts int, timeAttempts, retentionTime time.Duration, createdBy string
* @return *Flow
**/
func newFlow(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := &Flow{
		Tag:           tag,
		Version:       version,
		Name:          name,
		Description:   description,
		TpConsistency: TpConsistencyEventual,
		Steps:         make([]*Step, 0),
		CheckList:     make([]*CheckList, 0),
		CreatedBy:     createdBy,
		isDebug:       envar.GetBool("DEBUG", false),
	}
	logs.Logf(packageName, MSG_FLOW_CREATED, tag, version, name)
	flow.Step("Start", MSG_START_WORKFLOW, fn, stop)

	return flow
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Flow) serialize() ([]byte, error) {
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
func (s *Flow) ToJson() et.Json {
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
* setConfig
* @return error
**/
func (s *Flow) setConfig(format string, args ...any) {
	event.Publish(EVENT_WORKFLOW_SET, s.ToJson())
	if s.isDebug {
		logs.Logf(packageName, format, args...)
	}
}

/**
* Debug
* @return *Flow
**/
func (s *Flow) Debug() *Flow {
	s.isDebug = true
	return s
}

/**
* OnDone
* @param fn func(*Instance)
* @return *Flow
**/
func (s *Flow) OnDone(fn func(*Instance)) *Flow {
	s.onDone = fn
	return s
}

/**
* newStep
* @param name, description string, fn FnContext, stop bool
* @return *Step
**/
func (s *Flow) newStep(name, description string, fn FnContext, stop bool) *Step {
	step, _ := newStep(name, description, fn, stop)
	s.Steps = append(s.Steps, step)
	s.setConfig(MSG_INSTANCE_STEP_CREATED, len(s.Steps)-1, name, s.Tag)
	return step
}

/**
* Step
* @param name, description string, fn FnContext, stop bool
* @return *Flow
**/
func (s *Flow) Step(name, description string, fn FnContext, stop bool) *Flow {
	s.newStep(name, description, fn, stop)
	return s
}

/**
* StepWait
* @param name, description string, fn FnContext, timeAwait string, stop bool
* @return *Flow
**/
func (s *Flow) StepWait(name, description string, fn FnContext, timeAwait string, stop bool) *Flow {
	step := s.newStep(name, description, fn, stop)
	step.Kind = StepWait
	step.Spec = timeAwait
	return s
}

/**
* Rollback
* @params fn FnContext
* @return *Flow
**/
func (s *Flow) Rollback(fn FnContext) *Flow {
	n := len(s.Steps)
	step := s.Steps[n-1]
	step.rollbacks = fn
	s.setConfig(MSG_INSTANCE_ROLLBACK_CREATED, n-1, step.Name, s.Tag)

	return s
}

/**
* Consistency
* @param consistency TpConsistency
* @return *Flow
**/
func (s *Flow) Consistency(consistency TpConsistency) *Flow {
	s.TpConsistency = consistency
	s.setConfig(MSG_INSTANCE_CONSISTENCY, s.Tag, s.TpConsistency)

	return s
}

/**
* Resilence
* @param totalAttempts int, timeAttempts time.Duration
* @return *Flow
**/
func (s *Flow) Resilence(totalAttempts int, timeAttempts time.Duration, team string, level string) *Flow {
	s.TotalAttempts = totalAttempts
	s.TimeAttempts = timeAttempts
	s.Team = team
	s.Level = level
	s.setConfig(MSG_INSTANCE_RESILIENCE, s.Tag, totalAttempts, timeAttempts)

	return s
}

/**
* IfElse
* @param expression string, yesGoTo int, noGoTo int
* @return *Flow, error
**/
func (s *Flow) IfElse(expression string, yesGoTo int, noGoTo int) *Flow {
	n := len(s.Steps)
	step := s.Steps[n-1]
	step.ifElse(expression, yesGoTo, noGoTo)
	s.setConfig(MSG_INSTANCE_IFELSE, n-1, step.Name, expression, yesGoTo, noGoTo, s.Tag)

	return s
}

/**
* DefineCheckList
* @param tag string, description string, ok bool, data et.Json
* @return *Flow
**/
func (s *Flow) DefineCheckList(tag string, description string) *Flow {
	s.CheckList = append(s.CheckList, &CheckList{
		Tag:         tag,
		Description: description,
		Ok:          false,
		Data:        et.Json{},
	})
	return s
}

/**
* RemoveCheckList
* @param tag string
* @return *Flow
**/
func (s *Flow) RemoveCheckList(tag string) *Flow {
	idx := slices.IndexFunc(s.CheckList, func(check *CheckList) bool { return check.Tag == tag })
	if idx != -1 {
		s.CheckList = append(s.CheckList[:idx], s.CheckList[idx+1:]...)
	}

	return s
}
