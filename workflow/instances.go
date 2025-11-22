package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/vm"
)

type FlowStatus string

const (
	FlowStatusPending FlowStatus = "pending"
	FlowStatusRunning FlowStatus = "running"
	FlowStatusDone    FlowStatus = "done"
	FlowStatusFailed  FlowStatus = "failed"
)

type LoadInstanceFn func(id string) (*Instance, error)
type SaveInstanceFn func(instance *Instance) error
type DeleteInstanceFn func(id string) error

var (
	loadInstance LoadInstanceFn
	saveInstance SaveInstanceFn
	delInstance  DeleteInstanceFn
)

/**
* OnLoadInstance
* @param f LoadInstanceFn
* @return void
**/
func OnLoadInstance(f LoadInstanceFn) {
	loadInstance = f
}

/**
* OnSaveInstance
* @param f SaveInstanceFn
* @return void
**/
func OnSaveInstance(f SaveInstanceFn) {
	saveInstance = f
}

/**
* OnDeleteInstance
* @param f DeleteInstanceFn
* @return void
**/
func OnDeleteInstance(f DeleteInstanceFn) {
	delInstance = f
}

type Instance struct {
	*Flow
	workFlows  *WorkFlows           `json:"-"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Tag        string               `json:"tag"`
	Id         string               `json:"id"`
	CreatedBy  string               `json:"created_by"`
	UpdatedBy  string               `json:"updated_by"`
	Status     FlowStatus           `json:"status"`
	DoneAt     time.Time            `json:"done_at"`
	Current    int                  `json:"current"`
	Ctx        et.Json              `json:"ctx"`
	Ctxs       map[int]et.Json      `json:"ctxs"`
	PinnedData et.Json              `json:"pinned_data"`
	Results    map[int]*Result      `json:"results"`
	Tags       et.Json              `json:"tags"`
	Rollbacks  map[int]*Result      `json:"rollbacks"`
	WorkerHost string               `json:"worker_host"`
	vm         *vm.Vm               `json:"-"`
	done       bool                 `json:"-"`
	goTo       int                  `json:"-"`
	err        error                `json:"-"`
	resilence  *resilience.Instance `json:"-"`
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Instance) serialize() ([]byte, error) {
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
	bt, err := s.serialize()
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
* Save
* @return error
**/
func (s *Instance) Save() error {
	if saveInstance != nil {
		err := saveInstance(s)
		if err != nil {
			err = fmt.Errorf("saveInstance: error on save instanceId: %s, error: %v", s.Id, err)
			event.Publish(EVENT_ERROR, et.Json{
				"message": err.Error(),
			})
			return err
		}
	}
	event.Publish(EVENT_WORKFLOW_SET, s.ToJson())
	return nil
}

/**
* SetStatus
* @param status FlowStatus
* @return error
**/
func (s *Instance) SetStatus(status FlowStatus) error {
	save := func() error {
		return s.Save()
	}

	if s.Status == status {
		return save()
	}

	s.Status = status
	s.UpdatedAt = utility.NowTime()

	if s.Status == FlowStatusDone {
		s.DoneAt = s.UpdatedAt
		s.done = true
	}

	if s.Status == FlowStatusFailed {
		if s.resilence != nil && s.resilence.IsEnd() {
			s.done = true
		}

		errMsg := ""
		if s.err != nil {
			errMsg = s.err.Error()
		}
		logs.Errorf(MSG_INSTANCE_FAILED, s.Id, s.Tag, s.Status, s.Current, errMsg)
	} else {
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.Id, s.Tag, s.Status, s.Current)
	}

	return save()
}

/**
* SetStep
* @param val int
* @return error
**/
func (s *Instance) SetStep(val int) {
	s.Current = val
}

/**
* SetCtx
* @param ctx et.Json
**/
func (s *Instance) SetCtx(ctx et.Json) et.Json {
	for k, v := range ctx {
		s.Ctx[k] = v
	}

	s.Ctxs[s.Current] = ctx.Clone()

	return s.Ctx
}

/**
* SetPinedData
* @param key string, value interface{}
**/
func (s *Instance) SetPinnedData(key string, value interface{}) {
	s.PinnedData[key] = value
}

/**
* SetResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) SetResult(result et.Json, err error) (et.Json, error) {
	s.err = err
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}

	attempt := 0
	if s.resilence != nil {
		attempt = s.resilence.Attempt
	}

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
* SetTags
* @param tags et.Json
**/
func (s *Instance) SetTags(tags et.Json) {
	for k, v := range tags {
		s.Tags[k] = v
	}
}

/**
* setDone
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setDone(result et.Json, err error) (et.Json, error) {
	s.SetResult(result, err)
	s.SetStatus(FlowStatusDone)

	return result, err
}

/**
* setFailed
* @param result et.Json, err error
**/
func (s *Instance) setFailed(result et.Json, err error) (et.Json, error) {
	s.SetResult(result, err)
	s.SetStatus(FlowStatusFailed)

	return result, err
}

/**
* setStop
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setStop(result et.Json, err error) (et.Json, error) {
	s.SetResult(result, err)
	s.SetStep(s.Current + 1)
	s.SetStatus(FlowStatusPending)

	return result, err
}

/**
* setNext
* @return error
**/
func (s *Instance) setNext(result et.Json, err error) (et.Json, error) {
	s.SetResult(result, err)
	s.SetStep(s.Current + 1)
	s.SetStatus(s.Status)

	return result, err
}

/**
* setGoto
* @param step int, result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setGoto(step int, message string, result et.Json, err error) (et.Json, error) {
	s.SetResult(result, err)
	s.SetStep(step)
	s.goTo = -1
	s.SetStatus(s.Status)
	logs.Logf(packageName, MSG_INSTANCE_GOTO, s.Id, s.Tag, step, message)

	return result, err
}

/**
* run
* @param ctx et.Json, runerBy string
* @return et.Json, error
**/
func (s *Instance) run(ctx et.Json, runerBy string) (et.Json, error) {
	if s.Status == FlowStatusDone {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.Status == FlowStatusRunning {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING)
	} else if s.Current >= len(s.Steps) {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.Current < 0 {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.done {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	}

	s.UpdatedBy = runerBy
	var err error
	for s.Current < len(s.Steps) {
		ctx = s.SetCtx(ctx)
		step := s.Steps[s.Current]
		ctx, err = step.run(s, ctx)
		if err != nil {
			return s.rollback(ctx, err)
		}

		if s.done {
			return s.setDone(ctx, err)
		}

		if step.Stop {
			return s.setStop(ctx, err)
		}

		if s.goTo != -1 {
			s.setGoto(s.goTo, MSG_INSTANCE_GOTO_USER_DECISION, ctx, err)
			continue
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
* @param idx int
* @return et.Json, error
**/
func (s *Instance) rollback(result et.Json, err error) (et.Json, error) {
	s.setFailed(result, err)
	if s.TotalAttempts == 0 {
		return result, err
	} else if s.Status == FlowStatusDone {
		return result, fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.Status == FlowStatusPending {
		return result, fmt.Errorf(MSG_INSTANCE_PENDING)
	}

	if s.resilence == nil {
		description := fmt.Sprintf("flow: %s,  %s", s.Name, s.Description)
		s.resilence = resilience.AddCustom(s.Id, s.Tag, description, s.TotalAttempts, s.TimeAttempts, s.RetentionTime, s.Tags, s.Team, s.Level, s.run, s.Ctx)
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
			attempt := 0
			if s.resilence != nil {
				attempt = s.resilence.Attempt
			}
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
	s.SetStatus(s.Status)

	return nil
}

/**
* Done
* @return error
**/
func (s *Instance) Done() error {
	s.SetStatus(FlowStatusDone)

	return nil
}

/**
* Goto
* @param step int
* @return error
**/
func (s *Instance) Goto(step int) error {
	s.goTo = step
	s.SetStatus(s.Status)

	return nil
}
