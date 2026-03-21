package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

const (
	Pending Status = "pending"
	Running Status = "running"
	Done    Status = "done"
	Failed  Status = "failed"
	Loss    Status = "loss"
	Cancel  Status = "cancel"
)

var FlowStatusList map[Status]bool = map[Status]bool{
	Pending: true,
	Running: true,
	Done:    true,
	Failed:  true,
	Loss:    true,
	Cancel:  true,
}

type Instance struct {
	*Flow
	CreatedAt  time.Time                       `json:"created_at"`
	UpdatedAt  time.Time                       `json:"updated_at"`
	Tag        string                          `json:"tag"`
	ID         string                          `json:"id"`
	CreatedBy  string                          `json:"created_by"`
	UpdatedBy  string                          `json:"updated_by"`
	Current    int                             `json:"current"`
	Ctx        et.Json                         `json:"ctx"`
	Ctxs       map[int]et.Json                 `json:"ctxs"`
	Results    map[int]*Result                 `json:"results"`
	Rollbacks  map[int]*Result                 `json:"rollbacks"`
	Params     et.Json                         `json:"params"`
	Traces     []et.Json                       `json:"traces"`
	Status     Status                          `json:"status"`
	DoneAt     time.Time                       `json:"done_at"`
	Tags       et.Json                         `json:"tags"`
	WorkerHost string                          `json:"worker_host"`
	Resilence  map[string]*resilience.Instance `json:"resilence"`
	owner      *WorkFlow                       `json:"-"`
	done       bool                            `json:"-"`
	goTo       int                             `json:"-"`
	err        error                           `json:"-"`
	isNew      bool                            `json:"-"`
	isDebug    bool                            `json:"-"`
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Instance) Serialize() ([]byte, error) {
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
func (s *Instance) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result
}

/**
* ToString
* @return string
**/
func (s *Instance) ToString() string {
	return s.ToJson().ToString()
}

/**
* Save
* @return error
**/
func (s *Instance) Save() error {
	data := s.ToJson()
	event.Publish(EVENT_WORKFLOW_STATUS, data)

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.setInstance != nil {
		return s.owner.setInstance(s.ID, s.Tag, s)
	}

	return nil
}

/**
* setStatus
* @param status FlowStatus
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	s.Status = status
	s.UpdatedAt = timezone.Now()
	switch s.Status {
	case Done:
		s.DoneAt = s.UpdatedAt
		s.done = true
	case Failed:
		errMsg := ""
		if s.err != nil {
			errMsg = s.err.Error()
		}
		logs.Errorf(packageName, MSG_INSTANCE_FAILED, s.ID, s.Tag, s.Status, s.Current, errMsg)
	default:
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.Tag, s.Status, s.Current)
	}

	err := s.Save()
	if err != nil {
		return err
	}

	return nil
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) (et.Json, error) {
	s.err = err
	errMessage := ""
	if s.err != nil {
		errMessage = s.err.Error()
	}

	attempt := len(s.Resilence)
	res := &Result{
		Step:    s.Current,
		Ctx:     s.Ctx.Clone(),
		Attempt: attempt,
		Result:  result,
		Error:   errMessage,
	}
	s.Results[s.Current] = res

	return result, err
}

/**
* SetTag
* @param key string, value interface{}
* @return error
**/
func (s *Instance) SetTag(key string, value interface{}) error {
	s.Tags[key] = value
	return s.Save()
}

/**
* PutTag
* @param tags et.Json
* @return error
**/
func (s *Instance) PutTag(tags et.Json) error {
	for k, v := range tags {
		s.Tags[k] = v
	}
	return s.Save()
}

/**
* GetTag
* @param key string
* @return interface{}
**/
func (s *Instance) GetTag(key string) interface{} {
	return s.Tags[key]
}

/**
* SetParam
* @param key string, value interface{}
**/
func (s *Instance) SetParam(key string, value interface{}) (et.Json, error) {
	s.Params[key] = value
	err := s.Save()
	if err != nil {
		return s.Params, err
	}

	return s.Params, nil
}

/**
* PutParam
* @param value et.Json
* @return et.Json, error
**/
func (s *Instance) PutParam(value et.Json) (et.Json, error) {
	for k, v := range value {
		s.Params[k] = v
	}
	err := s.Save()
	if err != nil {
		return s.Params, err
	}

	return s.Params, nil
}

/**
* GetParam
* @param key string
* @return interface{}
**/
func (s *Instance) GetParam(key string) interface{} {
	return s.Params[key]
}

/**
* setTrace
* @param step int, ctx et.Json, err error
* @return error
**/
func (s *Instance) setTrace(step int, result et.Json, err error) error {
	ctx := s.getCtx(step)
	s.Traces = append(s.Traces, et.Json{
		"step":   step,
		"ctx":    ctx,
		"result": result,
		"error":  err,
	})
	er := s.Save()
	if er != nil {
		return er
	}

	return err
}

/**
* GetTraces
* @param idx int
* @return (et.Json, error)
**/
func (s *Instance) GetTraces(idx int) (et.Json, error) {
	if idx < 0 || idx >= len(s.Traces) {
		return et.Json{}, fmt.Errorf("trace not found")
	}

	return s.Traces[idx], nil
}

/**
* GetTraceByStep
* @params step int
* @return []et.Json
**/
func (s *Instance) GetTraceByStep(step int) []et.Json {
	result := []et.Json{}
	for _, trace := range s.Traces {
		if trace["step"] == step {
			result = append(result, trace)
		}
	}

	return result
}

/**
* SetCheckList
* @param tag string, ok bool, data et.Json
* @return error
**/
func (s *Instance) SetCheckList(tag string, ok bool, data et.Json) error {
	idx := slices.IndexFunc(s.CheckList, func(check *CheckList) bool { return check.Tag == tag })
	if idx != -1 {
		s.CheckList[idx].Ok = ok
		s.CheckList[idx].Data = data
		return s.Save()
	}

	return fmt.Errorf("check list not found")
}

/**
* @param ctx et.Json
* @return error
**/
func (s *Instance) getCtx(idx int) et.Json {
	result, ok := s.Ctxs[idx]
	if !ok {
		return et.Json{}
	}

	return result
}

/**
* setCtx
* @param ctx et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	for k, v := range ctx {
		s.Ctx[k] = v
	}

	s.Ctxs[s.Current] = ctx.Clone()

	return s.Ctx
}

/**
* SetCtx
* @param ctx et.Json
* @return error
**/
func (s *Instance) SetCtx(ctx et.Json) error {
	s.setCtx(ctx)
	return s.Save()
}

/**
* setDone
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setDone(result et.Json, err error) (et.Json, error) {
	s.setResult(result, err)
	errStatus := s.setStatus(Done)
	if errStatus != nil {
		return result, errStatus
	}

	return result, err
}

/**
* setFailed
* @param result et.Json, err error
**/
func (s *Instance) setFailed(result et.Json, err error) error {
	s.setResult(result, err)
	errStatus := s.setStatus(Failed)
	if errStatus != nil {
		return errStatus
	}

	return nil
}

/**
* setStop
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setStop(result et.Json, err error) (et.Json, error) {
	s.setResult(result, err)
	s.Current++
	errStatus := s.setStatus(Pending)
	if errStatus != nil {
		return result, errStatus
	}

	return result, err
}

/**
* setNext
* @return error
**/
func (s *Instance) setNext(result et.Json, err error) error {
	s.setResult(result, err)
	s.Current++
	errStatus := s.setStatus(s.Status)
	if errStatus != nil {
		return errStatus
	}

	return nil
}

/**
* setGoto
* @param step int, result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setGoto(step int, message string, result et.Json, err error) error {
	if step == -1 {
		return nil
	}

	s.setResult(result, err)
	s.Current = step
	s.goTo = -1
	errStatus := s.setStatus(s.Status)
	if errStatus != nil {
		return errStatus
	}

	logs.Logf(packageName, MSG_INSTANCE_GOTO, s.ID, s.Tag, step, message)
	return nil
}

/**
* startResilence
* @return bool
**/
func (s *Instance) startResilence() bool {
	if s.TotalAttempts == 0 {
		return false
	}

	description := fmt.Sprintf("flow: %s,  %s", s.Name, s.Description)
	instance := s.owner.resilience.Run(s.Tag, description, s.TotalAttempts, s.TimeAttempts, s.Tags, s.Team, s.Level, s.run, s.Ctx)
	s.Resilence[instance.ID] = instance
	return true
}

/**
* run
* @param ctx et.Json
* @return et.Json, error
**/
func (s *Instance) run(ctx et.Json) (et.Json, error) {
	var err error
	defer func() {
		s.setTrace(s.Current, ctx, err)
	}()

	if s.Status == Done {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
		return et.Json{}, err
	} else if s.Status == Running && s.isNew {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
		return et.Json{}, err
	} else if s.Status == Cancel {
		err = fmt.Errorf(MSG_INSTANCE_CANCEL, s.ID)
		return et.Json{}, err
	} else if s.Status == Loss {
		err = fmt.Errorf(MSG_INSTANCE_LOSS, s.ID)
		return et.Json{}, err
	}

	for s.Current < len(s.Steps) {
		step := s.Steps[s.Current]
		ctx = s.setCtx(ctx)
		ctx, err = step.run(s, ctx)
		if err != nil {
			return s.rollback(ctx, err)
		}

		if s.done {
			return s.setDone(ctx, err)
		}

		if s.goTo != -1 {
			s.setGoto(s.goTo, MSG_INSTANCE_GOTO_USER_DECISION, ctx, err)
			continue
		}

		if step.Stop {
			return s.setStop(ctx, err)
		}

		if step.Expression != "" {
			ok, err := step.evaluate(ctx, s)
			if err != nil {
				return s.rollback(ctx, err)
			}

			if ok {
				s.setGoto(step.YesGoTo, MSG_INSTANCE_EXPRESSION_TRUE, ctx, err)
			} else {
				s.setGoto(step.NoGoTo, MSG_INSTANCE_EXPRESSION_FALSE, ctx, err)
			}
		}

		if s.Current == len(s.Steps)-1 {
			return s.setDone(ctx, err)
		}

		s.setNext(ctx, err)
	}

	return ctx, err
}

/**
* rollback
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) rollback(result et.Json, err error) (et.Json, error) {
	s.setFailed(result, err)
	if s.startResilence() {
		return result, err
	}

	if s.Status == Done {
		return result, fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
	} else if s.Status == Running {
		return result, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
	} else if s.Status == Pending {
		return result, fmt.Errorf(MSG_INSTANCE_PENDING, s.ID)
	}

	for i := s.Current - 1; i >= 0; i-- {
		logs.Logf(packageName, MSG_INSTANCE_ROLLBACK_STEP, i)
		step := s.Steps[i]
		if step == nil {
			continue
		}

		if step.rollbacks == nil {
			continue
		}

		if s.Ctxs[i] == nil {
			continue
		}

		ctx := s.Ctxs[i].Clone()
		result, err = step.rollbacks(s, ctx)
		if err != nil {
			attempt := len(s.Resilence)
			s.Rollbacks[i] = &Result{
				Step:    i,
				Ctx:     ctx,
				Attempt: attempt,
				Result:  result,
				Error:   err.Error(),
			}

			if s.TpConsistency == TpConsistencyStrong {
				return result, err
			}
		}
	}

	return result, err
}

/**
* Stop
* @return error
**/
func (s *Instance) Stop() error {
	s.Steps[s.Current].Stop = true
	return s.setStatus(s.Status)
}

/**
* Done
* @return error
**/
func (s *Instance) Done() error {
	return s.setStatus(Done)
}

/**
* Goto
* @param step int
* @return error
**/
func (s *Instance) Goto(step int) error {
	s.goTo = step
	return s.setStatus(s.Status)
}

/**
* SetStatus
* @param status Status
* @return error
**/
func (s *Instance) SetStatus(status Status) error {
	return s.setStatus(status)
}
