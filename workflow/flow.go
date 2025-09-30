package workflow

import (
	"encoding/json"
	"os"
	"time"

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

type FnContext func(flow *Instance, ctx et.Json) (et.Json, error)

type Flow struct {
	Tag           string        `json:"tag"`
	Version       string        `json:"version"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	TotalAttempts int           `json:"total_attempts"`
	TimeAttempts  time.Duration `json:"time_attempts"`
	RetentionTime time.Duration `json:"retention_time"`
	Steps         []*Step       `json:"steps"`
	TpConsistency TpConsistency `json:"tp_consistency"`
	Team          string        `json:"team"`
	Level         string        `json:"level"`
	CreatedBy     string        `json:"created_by"`
	isDebug       bool          `json:"-"`
}

/**
* newFlow
* @param tag, version, name, description string, createdBy string
* @return *Flow
**/
func newFlow(tag, version, name, description string, createdBy string) *Flow {
	return &Flow{
		Tag:           tag,
		Version:       version,
		Name:          name,
		Description:   description,
		TpConsistency: TpConsistencyEventual,
		Steps:         make([]*Step, 0),
		CreatedBy:     createdBy,
	}
}

/**
* newFlowFn
* @param tag, version, name, description string, fn FnContext, stop bool, createdBy string
* @return *Flow
**/
func newFlowFn(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := newFlow(tag, version, name, description, createdBy)
	logs.Logf(packageName, MSG_FLOW_CREATED, tag, version, name)
	flow.StepFn("Start", MSG_START_WORKFLOW, fn, stop)

	return flow
}

/**
* newFlowDefinition
* @param tag, version, name, description string, definition string, totalAttempts int, timeAttempts, retentionTime time.Duration, createdBy string
* @return *Flow
**/
func newFlowDefinition(tag, version, name, description string, definition string, stop bool, createdBy string) *Flow {
	flow := newFlow(tag, version, name, description, createdBy)
	logs.Logf(packageName, MSG_FLOW_CREATED, tag, version, name)
	flow.StepDefinition("Start", MSG_START_WORKFLOW, definition, stop)

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
	logs.Logf(packageName, format, args...)
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
* StepFn
* @param name, description string, fn FnContext, retries, retryDelay int, stop bool
* @return *Fn
**/
func (s *Flow) StepFn(name, description string, fn FnContext, stop bool) *Flow {
	result, _ := newStepFn(name, description, fn, stop)
	s.Steps = append(s.Steps, result)
	s.setConfig(MSG_INSTANCE_STEP_CREATED, len(s.Steps)-1, name, s.Tag)

	return s
}

/**
* StepDefinition
* @param name, description string, definition string, stop bool
* @return *Flow
**/
func (s *Flow) StepDefinition(name, description string, definition string, stop bool) *Flow {
	result, _ := newStepDefinition(name, description, definition, stop)
	s.Steps = append(s.Steps, result)
	s.setConfig(MSG_INSTANCE_STEP_CREATED, len(s.Steps)-1, name, s.Tag)

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
	retentionTime := time.Duration(s.TotalAttempts * int(timeAttempts))
	if s.RetentionTime < retentionTime {
		s.RetentionTime = retentionTime
	}
	s.Team = team
	s.Level = level
	s.setConfig(MSG_INSTANCE_RESILIENCE, s.Tag, totalAttempts, timeAttempts, retentionTime)

	return s
}

/**
* Retention
* @param retentionTime time.Duration
* @return *Flow
**/
func (s *Flow) Retention(retentionTime time.Duration) *Flow {
	s.RetentionTime = retentionTime
	s.setConfig(MSG_INSTANCE_RETENTION, s.Tag, retentionTime)

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
