package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
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
* ToJson
* @return (et.Json, error)
**/
func (s *Instance) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result, nil
}

/**
* ToString
* @return string
**/
func (s *Instance) ToString() string {
	result, err := s.ToJson()
	if err != nil {
		return ""
	}

	return result.ToString()
}

/**
* Save
* @return error
**/
func (s *Instance) Save() error {
	data, err := s.ToJson()
	if err != nil {
		return err
	}

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.store != nil {
		return s.owner.store.Set(s.ID, s.Tag, s)
	}

	return nil
}

/**
* SetTag
* @param key string, value interface{}
* @return (et.Json, error)
**/
func (s *Instance) SetTag(key string, value interface{}) (et.Json, error) {
	s.Tags[key] = value
	err := s.Save()
	if err != nil {
		return s.Tags, err
	}
	return s.Tags, nil
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
* @return error
**/
func (s *Instance) PutParam(value et.Json) error {
	for k, v := range value {
		s.Params[k] = v
	}
	return s.Save()
}

/**
* setTrace
* @param step int, result et.Json, err error
* @return error
**/
func (s *Instance) setTrace(step int, result et.Json, err error) error {
	s.Traces = append(s.Traces, et.Json{
		"step":   step,
		"ctx":    s.Ctx.Clone(),
		"result": result,
		"error":  err,
	})
	return s.Save()
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
* setStatus
* @param status FlowStatus
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	if s.Status == status {
		return nil
	}

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
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json, step int) et.Json {
	for k, v := range ctx {
		s.Ctx[k] = v
	}

	s.Ctxs[step] = s.Ctx.Clone()
	return s.Ctx
}

/**
* getCtx
* @param idx int
* @return et.Json
**/
func (s *Instance) getCtx(idx int) et.Json {
	result, ok := s.Ctxs[idx]
	if !ok {
		return et.Json{}
	}

	return result
}

/**
* setDone
* @param result et.Json
* @return et.Json, error
**/
func (s *Instance) setDone(result et.Json) (et.Json, error) {
	s.setResult(result, nil)
	err := s.setStatus(Done)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* setFailed
* @param err error
**/
func (s *Instance) setFailed(err error) error {
	s.setResult(et.Json{}, err)
	err = s.setStatus(Failed)
	if err != nil {
		return err
	}

	return nil
}

/**
* setStop
* @param result et.Json
* @return (et.Json, error)
**/
func (s *Instance) setStop(result et.Json) (et.Json, error) {
	s.setResult(result, nil)
	s.Current++
	err := s.setStatus(Pending)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* setNext
* @return error
**/
func (s *Instance) setNext(result et.Json) error {
	s.setResult(result, nil)
	s.Current++
	err := s.setStatus(Running)
	if err != nil {
		return err
	}

	return nil
}

/**
* setGoto
* @param step int, message string, result et.Json
* @return et.Json, error
**/
func (s *Instance) setGoto(step int, message string, result et.Json) (et.Json, error) {
	if step == -1 {
		return result, nil
	}

	s.setResult(result, nil)
	s.Current = step
	s.goTo = -1
	err := s.setStatus(Running)
	if err != nil {
		return result, err
	}

	logs.Logf(packageName, MSG_INSTANCE_GOTO, s.ID, s.Tag, step, message)
	return result, nil
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
		ctx = s.setCtx(ctx, s.Current)
		result, err := step.run(s, ctx)
		if err != nil {
			return s.rollback(ctx, err)
		}

		if s.done {
			return s.setDone(result)
		}

		if step.Stop {
			return s.setStop(result)
		}

		if s.goTo != -1 {
			ctx, err = s.setGoto(s.goTo, MSG_INSTANCE_GOTO_USER_DECISION, result)
			if err != nil {
				return result, err
			}
			continue
		}

		if step.Condition != nil {
			ok, err := step.evaluate(result, s)
			if err != nil {
				return et.Json{}, err
			}

			if ok {
				ctx, err = s.setGoto(step.Condition.YesTo, MSG_INSTANCE_EXPRESSION_TRUE, result)
				if err != nil {
					return result, err
				}
			} else {
				ctx, err = s.setGoto(step.Condition.NoTo, MSG_INSTANCE_EXPRESSION_FALSE, result)
				if err != nil {
					return result, err
				}
			}
			continue
		}

		if s.Current == len(s.Steps)-1 {
			return s.setDone(result)
		}

		s.setNext(result)
	}

	return ctx, err
}

/**
* rollback
* @param ctx et.Json, err error
* @return et.Json, error
**/
func (s *Instance) rollback(ctx et.Json, err error) (et.Json, error) {
	s.setFailed(err)
	if s.startResilence() {
		return ctx, err
	}

	if s.Status == Done {
		return ctx, fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
	} else if s.Status == Running {
		return ctx, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
	} else if s.Status == Pending {
		return ctx, fmt.Errorf(MSG_INSTANCE_PENDING, s.ID)
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
		result, err := step.rollbacks(s, ctx)
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
				return ctx, err
			}
		}
	}

	return ctx, err
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
