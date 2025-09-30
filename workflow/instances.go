package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/jdb/jdb"
)

type FlowStatus string

const (
	FlowStatusPending FlowStatus = "pending"
	FlowStatusRunning FlowStatus = "running"
	FlowStatusDone    FlowStatus = "done"
	FlowStatusFailed  FlowStatus = "failed"
)

type Instance struct {
	*Flow
	workFlows  *WorkFlows           `json:"-"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Id         string               `json:"id"`
	CreatedBy  string               `json:"created_by"`
	Current    int                  `json:"current"`
	Ctx        et.Json              `json:"ctx"`
	Ctxs       map[int]et.Json      `json:"ctxs"`
	Results    map[int]*Result      `json:"results"`
	Rollbacks  map[int]*Result      `json:"rollbacks"`
	Status     FlowStatus           `json:"status"`
	DoneAt     time.Time            `json:"done_at"`
	Tags       et.Json              `json:"tags"`
	WorkerHost string               `json:"worker_host"`
	Params     et.Json              `json:"params"`
	done       bool                 `json:"-"`
	goTo       int                  `json:"-"`
	err        error                `json:"-"`
	resilence  *resilience.Instance `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	steps := make([]et.Json, len(s.Steps))
	for i, step := range s.Steps {
		j := step.ToJson()
		j.Set(jdb.KEY, i)
		steps[i] = j
	}

	resilence := et.Json{}
	if s.resilence != nil {
		resilence = s.resilence.ToJson()
	}

	result := et.Json{
		"id":             s.Id,
		"tag":            s.Tag,
		"version":        s.Version,
		"name":           s.Name,
		"description":    s.Description,
		"current":        s.Current,
		"total_attempts": s.TotalAttempts,
		"time_attempts":  s.TimeAttempts,
		"retention_time": s.RetentionTime,
		"ctx":            s.Ctx,
		"steps":          steps,
		"ctxs":           s.Ctxs,
		"results":        s.Results,
		"rollbacks":      s.Rollbacks,
		"resilence":      resilence,
		"tp_consistency": s.TpConsistency,
		"created_at":     s.CreatedAt,
		"updated_at":     s.UpdatedAt,
		"done_at":        s.DoneAt,
		"status":         s.Status,
		"worker_host":    s.WorkerHost,
		"params":         s.Params,
	}

	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	event.Publish(EVENT_WORKFLOW_STATUS, s.ToJson())
	bt, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if s.RetentionTime <= 0 {
		s.RetentionTime = 24 * time.Hour
	}

	cache.Set(s.Id, string(bt), s.RetentionTime)

	return nil
}

/**
* setStatus
* @param status FlowStatus
* @return error
**/
func (s *Instance) setStatus(status FlowStatus) error {
	if s.Status == status {
		err := s.save()
		if err != nil {
			return fmt.Errorf("setStatus: error al guardar el estado de la instancia: %v", err)
		}

		return nil
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
		logs.Errorf(packageName, MSG_INSTANCE_FAILED, s.Id, s.Tag, s.Status, s.Current, errMsg)
	} else {
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.Id, s.Tag, s.Status, s.Current)
	}

	err := s.save()
	if err != nil {
		return fmt.Errorf("setStatus: error al guardar el estado de la instancia: %v", err)
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
* setTags
* @param tags et.Json
**/
func (s *Instance) setTags(tags et.Json) {
	for k, v := range tags {
		s.Tags[k] = v
	}
}

/**
* SetParam
* @param key string, value interface{}
**/
func (s *Instance) SetParam(key string, value interface{}) {
	s.Params[key] = value
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
* setDone
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setDone(result et.Json, err error) (et.Json, error) {
	s.setResult(result, err)
	s.setStatus(FlowStatusDone)

	return result, err
}

/**
* setFailed
* @param result et.Json, err error
**/
func (s *Instance) setFailed(result et.Json, err error) {
	s.setResult(result, err)
	s.setStatus(FlowStatusFailed)
}

/**
* setStop
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setStop(result et.Json, err error) (et.Json, error) {
	s.setResult(result, err)
	s.Current++
	s.setStatus(FlowStatusPending)

	return result, err
}

/**
* setNext
* @return error
**/
func (s *Instance) setNext(result et.Json, err error) {
	s.setResult(result, err)
	s.Current++
	s.setStatus(s.Status)
}

/**
* setGoto
* @param step int, result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setGoto(step int, message string, result et.Json, err error) {
	s.setResult(result, err)
	s.Current = step
	s.goTo = -1
	s.setStatus(s.Status)
	logs.Logf(packageName, MSG_INSTANCE_GOTO, s.Id, s.Tag, step, message)
}

/**
* startResilence
* @return bool
**/
func (s *Instance) startResilence() bool {
	if s.TotalAttempts == 0 {
		return false
	}

	if s.resilence != nil {
		return !s.resilence.IsFailed()
	}

	description := fmt.Sprintf("flow: %s,  %s", s.Name, s.Description)
	s.resilence = resilience.AddCustom(s.Id, s.Tag, description, s.TotalAttempts, s.TimeAttempts, s.RetentionTime, s.Tags, s.Team, s.Level, s.run, s.Ctx)
	return true
}

/**
* run
* @param ctx et.Json
* @return et.Json, error
**/
func (s *Instance) run(ctx et.Json) (et.Json, error) {
	if s.Status == FlowStatusDone {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.Status == FlowStatusRunning {
		return s.ToJson(), fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING)
	}

	var err error
	for s.Current < len(s.Steps) {
		ctx = s.setCtx(ctx)
		step := s.Steps[s.Current]
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
* @param idx int
* @return et.Json, error
**/
func (s *Instance) rollback(result et.Json, err error) (et.Json, error) {
	if s.startResilence() {
		return result, err
	}

	if s.Status == FlowStatusDone {
		return result, fmt.Errorf(MSG_INSTANCE_ALREADY_DONE)
	} else if s.Status == FlowStatusRunning {
		return result, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING)
	} else if s.Status == FlowStatusPending {
		return result, fmt.Errorf(MSG_INSTANCE_PENDING)
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
	s.setStatus(s.Status)

	return nil
}

/**
* Done
* @return error
**/
func (s *Instance) Done() error {
	s.setStatus(FlowStatusDone)

	return nil
}

/**
* Goto
* @param step int
* @return error
**/
func (s *Instance) Goto(step int) error {
	s.goTo = step
	s.setStatus(s.Status)

	return nil
}
