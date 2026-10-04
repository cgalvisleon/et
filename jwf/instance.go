package jwf

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

func (s Status) Str() string {
	return string(s)
}

const (
	SYSTEM   Status = "system"
	ACTIVE   Status = "active"
	ARCHIVED Status = "archived"
	// Instance Status
	CREATED  Status = "created"
	PENDING  Status = "pending"
	RUNNING  Status = "running"
	ROLLBACK Status = "rollback"
	DONE     Status = "done"
	FAILED   Status = "failed"
	CANCEL   Status = "cancel"
	// Step Status (inside an instance)
	COMPLETED Status = "completed"
)

var (
	ErrorInstanceNotFound                 = errors.New(MSG_INSTANCE_NOT_FOUND)
	FlowStatusList        map[Status]bool = map[Status]bool{
		CREATED:  true,
		PENDING:  true,
		RUNNING:  true,
		ROLLBACK: true,
		DONE:     true,
		FAILED:   true,
		CANCEL:   true,
	}
)

type Current struct {
	SourceId   string `json:"source_id"`
	TargetId   string `json:"target_id"`
	ErrorId    string `json:"error_id"`
	Index      int    `json:"index"`
	IsFinished bool   `json:"is_finished"`
}

type Instance struct {
	StartedAt   time.Time                  `json:"started_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	DoneAt      time.Time                  `json:"done_at"`
	ID          string                     `json:"id"`
	FlowId      string                     `json:"flow_id"`
	FlowTag     string                     `json:"flow_tag"`
	FlowVersion string                     `json:"flow_version"`
	Code        string                     `json:"code"`
	Name        string                     `json:"name"`
	Status      Status                     `json:"status"`
	Ctx         et.Json                    `json:"ctx"`
	Params      et.Json                    `json:"params"`
	Steps       []et.Json                  `json:"steps"`
	Tags        et.Json                    `json:"tags"`
	Trigger     *Trigger                   `json:"trigger"`
	Current     *Connection                `json:"current"`
	Step        *Step                      `json:"step"`
	StepId      string                     `json:"step_id"`
	IsDone      bool                       `json:"is_done"`
	IsEnd       bool                       `json:"is_end"`
	UserId      string                     `json:"user_id"`
	Error       string                     `json:"error"`
	stop        bool                       `json:"-"`
	isDebug     bool                       `json:"-"`
	isChanged   bool                       `json:"-"`
	flow        *Flow                      `json:"-"`
	bindings    map[string]interface{}     `json:"-"`
	resilience  *resilience.Resilience     `json:"-"`
	goTo        int                        `json:"-"`
	onChange    []func(data et.Json) error `json:"-"`
	mu          sync.Mutex                 `json:"-"`
}

/**
* NewInstance
* @return *Instance
**/
func (s *Flow) NewInstance(id, code, name string, tags et.Json, trigger *Trigger, userId string) *Instance {
	now := timezone.Now()
	id = reg.GetUUID(id)
	result := &Instance{
		StartedAt:   now,
		ID:          id,
		FlowId:      s.ID,
		FlowTag:     s.Tag,
		FlowVersion: s.Version,
		Code:        code,
		Name:        name,
		Status:      CREATED,
		Params:      et.Json{},
		Ctx:         et.Json{},
		Steps:       make([]et.Json, 0),
		Tags:        tags,
		Trigger:     trigger,
		IsDone:      false,
		IsEnd:       false,
		UserId:      userId,
		flow:        s,
		bindings:    make(map[string]interface{}),
		onChange:    make([]func(data et.Json) error, 0),
		mu:          sync.Mutex{},
	}
	result.up()
	for name, binding := range s.bindings {
		result.SetBinding(name, binding)
	}
	return result
}

/**
* LoadInstance
* @param def et.Json
* @return *Instance, error
**/
func (s *Flow) LoadInstance(def et.Json) (*Instance, error) {
	bt, err := def.ToByte()
	if err != nil {
		return nil, err
	}

	var result *Instance
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	result.flow = s
	result.up()
	for name, binding := range s.bindings {
		result.SetBinding(name, binding)
	}
	return result, nil
}

/**
* up
* @return *Instance
**/
func (s *Instance) up() *Instance {
	if s.Steps == nil {
		s.Steps = make([]et.Json, 0)
	}
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	s.wrapper()
	return s
}

/**
* SetBinding
* @param key string, value interface{}
* @return *Instance
**/
func (s *Instance) SetBinding(key string, value interface{}) *Instance {
	s.bindings[key] = value
	return s
}

/**
* setGoto
* @param idx int
**/
func (s *Instance) setGoto(idx int) {
	s.goTo = idx
}

/**
* wrapper
* @param step *Step
**/
func (s *Instance) wrapper() {
	s.bindings["goTo"] = func(idx int) {
		if s.Step == nil {
			return
		}
		if idx < 0 || idx >= s.Step.Outputs {
			return
		}
		s.setGoto(idx)
	}
	s.bindings["ctx"] = map[string]interface{}{
		"set": func(data et.Json) {
			maps.Copy(s.Params, data)
		},
		"get": func(keys ...string) interface{} {
			return s.Params.Get(keys...)
		},
		"str": func(keys ...string) string {
			return s.Params.Str(keys...)
		},
		"int": func(keys ...string) int {
			return s.Params.Int(keys...)
		},
		"int64": func(keys ...string) int64 {
			return s.Params.Int64(keys...)
		},
		"num": func(keys ...string) float64 {
			return s.Params.Num(keys...)
		},
		"bool": func(keys ...string) bool {
			return s.Params.Bool(keys...)
		},
		"time": func(keys ...string) time.Time {
			return s.Params.Time(keys...)
		},
		"json": func(key string) et.Json {
			return s.Params.Json(key)
		},
		"array": func(key string) []interface{} {
			return s.Params.Array(key)
		},
		"arrayStr": func(key string) []string {
			return s.Params.ArrayStr(key)
		},
		"arrayInt": func(key string) []int {
			return s.Params.ArrayInt(key)
		},
		"arrayInt64": func(key string) []int64 {
			return s.Params.ArrayInt64(key)
		},
		"arrayJson": func(key string) []et.Json {
			return s.Params.ArrayJson(key)
		},
	}
}

/**
* push
* @return error
**/
func (s *Instance) push() {
	stepId := s.StepId
	if stepId == "" {
		stepId = "unknown"
	}

	if s.Error != "" {
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, stepId, s.Error)
	} else {
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.FlowId, stepId, s.Status)
	}

	for _, fn := range s.onChange {
		err := fn(s.ref())
		if err != nil {
			s.Error = err.Error()
			return
		}
	}
}

/**
* OnChange
* @param fn func(instance *Instance) error
* @return *Instance
**/
func (s *Instance) OnChange(fn func(data et.Json) error) *Instance {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	s.onChange = append(s.onChange, fn)
	return s
}

/**
* Ref
* @return et.Json
**/
func (s *Instance) ref() et.Json {
	current := any(nil)
	if s.Step != nil {
		current = et.Json{
			"step_id": s.Step.ID,
			"kind":    s.Step.Kind,
			"version": s.Step.Version,
			"status":  s.Step.Status,
			"title":   s.Step.Title,
		}
	}

	return et.Json{
		"id":           s.ID,
		"code":         s.Code,
		"flow_id":      s.FlowId,
		"flow_version": s.FlowVersion,
		"status":       s.getStatus(),
		"started_at":   timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at":   timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":      timezone.Format(s.DoneAt, timezone.RFC3339),
		"current":      current,
		"trigger":      s.Trigger,
		"params":       s.Params,
		"ctx":          s.Ctx,
		"steps":        s.Steps,
		"tags":         s.Tags,
		"step_id":      s.StepId,
		"is_done":      s.IsDone,
		"is_end":       s.IsEnd,
		"error":        nilIfEmpty(s.Error),
		"user_id":      s.UserId,
	}
}

/**
* nilIfEmpty: Returns nil for an empty string, so it is encoded as JSON null.
* @param value string
* @return any
**/
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	return et.Json{
		"started_at":   timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at":   timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":      timezone.Format(s.DoneAt, timezone.RFC3339),
		"id":           s.ID,
		"flow_id":      s.FlowId,
		"flow_tag":     s.FlowTag,
		"flow_version": s.FlowVersion,
		"code":         s.Code,
		"name":         s.Name,
		"status":       s.Status,
		"ctx":          s.Ctx,
		"params":       s.Params,
		"steps":        s.Steps,
		"tags":         s.Tags,
		"trigger":      s.Trigger,
		"current":      s.Current,
		"step":         s.Step,
		"step_id":      s.StepId,
		"is_done":      s.IsDone,
		"is_end":       s.IsEnd,
		"error":        s.Error,
		"user_id":      s.UserId,
	}
}

/**
* ToString
* @return string
**/
func (s *Instance) ToString() string {
	return s.ToJson().ToString()
}

/**
* getStatus
* @return Status
**/
func (s *Instance) getStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status) {
	curStatus := s.getStatus()
	if curStatus == status {
		return
	}

	s.UpdatedAt = timezone.Now()
	s.mu.Lock()
	s.Status = status
	s.mu.Unlock()
	switch status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	case FAILED, CANCEL:
		s.DoneAt = s.UpdatedAt
	}
	s.push()
}

/**
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	for k, v := range ctx {
		s.Ctx[k] = v
	}

	stepId := s.StepId
	if stepId == "" {
		return ctx
	}

	cp := et.Json{}
	maps.Copy(cp, ctx)
	s.Steps = append(s.Steps, et.Json{
		"step_id": stepId,
		"status":  s.getStatus(),
		"ctx":     cp,
		"result":  et.Json{},
		"error":   et.Json{},
	})

	s.push()
	return s.Ctx
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) *Instance {
	errMessage := et.Json{}
	if err != nil {
		errMessage = et.Json{
			"message": err.Error(),
		}
	}

	stepId := s.StepId
	if stepId == "" {
		return s
	}

	cp := et.Json{}
	maps.Copy(cp, result)
	for _, step := range s.Steps {
		if step.Str("step_id") == stepId {
			step["status"] = s.getStatus()
			step["result"] = cp
			step["error"] = errMessage
			break
		}
	}

	s.push()
	return s
}

/**
* setError
* @param err error
* @return error
**/
func (s *Instance) setError(err error) {
	if err != nil {
		s.Error = err.Error()
	}
	s.push()
}

/**
* setTag
* @param tags et.Json
* @return et.Json
**/
func (s *Instance) setTag(tags et.Json) et.Json {
	maps.Copy(s.Tags, tags)
	s.push()
	return s.Tags
}

/**
* setParams
* @param params et.Json
* @return et.Json
**/
func (s *Instance) SetParams(params et.Json) et.Json {
	maps.Copy(s.Params, params)
	s.push()
	return s.Params
}

/**
* Done
* @return *Instance
**/
func (s *Instance) Done() *Instance {
	s.setStatus(DONE)
	return s
}

/**
* setStep
* @param step *Step
**/
func (s *Instance) setEnd(step *Step, stepId string) bool {
	s.mu.Lock()
	s.Step = step
	s.StepId = stepId
	s.goTo = 0
	s.Current = nil
	s.IsEnd = true
	ok := s.Step != nil
	s.mu.Unlock()
	s.push()
	return ok
}

/**
* setCurrent
* @param step *Step
* @return error
**/
func (s *Instance) setCurrent(connection *Connection) bool {
	s.mu.Lock()
	s.Current = connection
	s.Step, _ = s.flow.getStep(connection.Source.StepId)
	s.StepId = connection.Source.StepId
	s.goTo = 0
	ok := s.Step != nil
	s.mu.Unlock()
	s.push()
	return ok
}

/**
* isStop
* @return bool
**/
func (s *Instance) isStop() bool {
	if s.Step != nil && s.Step.Stop {
		return true
	}

	if s.stop {
		return true
	}

	key := fmt.Sprintf("instance:%s:stop", s.ID)
	stop, _, err := cache.GetBool(key, false)
	if err != nil {
		return false
	}

	return stop
}

/**
* next
* @return bool
**/
func (s *Instance) next() bool {
	if s.isStop() {
		return false
	}

	if s.IsDone {
		return false
	}

	if s.IsEnd {
		s.Done()
		return false
	}

	status := s.getStatus()
	if status == CANCEL {
		return false
	}

	if s.Current == nil {
		connection, exists := s.flow.getOutput(s.Trigger.StartId, s.goTo)
		if !exists {
			return false
		}
		return s.setCurrent(connection)
	} else {
		connection, exists := s.flow.getOutput(s.Current.Target.StepId, s.goTo)
		if !exists {
			step, exists := s.flow.getStep(s.Current.Target.StepId)
			if !exists {
				return false
			}
			return s.setEnd(step, s.Current.Target.StepId)
		}
		return s.setCurrent(connection)
	}
}

/**
* run
* @param ctx, tags et.Json, await bool
* @return et.Json, error
**/
func (s *Instance) Run(ctx et.Json, await bool) (et.Json, error) {
	var err error
	var result et.Json
	defer func() {
		if err != nil {
			s.setError(err)
		}
		result["instance_wf"] = et.Json{
			"instance_id": s.ID,
			"flow_id":     s.FlowId,
			"flow_tag":    s.FlowTag,
			"code":        s.Code,
			"name":        s.Name,
			"status":      RUNNING,
			"ctx":         ctx,
		}
	}()

	status := s.getStatus()
	if status == DONE {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
		return et.Json{}, err
	} else if status == RUNNING {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
		return et.Json{}, err
	} else if status == ROLLBACK {
		err = fmt.Errorf(MSG_INSTANCE_ROLLBACK, s.ID)
		return et.Json{}, err
	} else if status == CANCEL {
		err = fmt.Errorf(MSG_INSTANCE_CANCEL, s.ID)
		return et.Json{}, err
	}

	runing := func() (et.Json, error) {
		for s.next() {
			step := s.Step
			if step == nil {
				return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
			}

			step.OnStatus(func(status Status) {
				s.setStatus(status)
			})

			ctx = s.setCtx(ctx)
			result, err = step.Run(s, ctx)
			if err != nil {
				result, err = s.runResilence(ctx, await, err)
				if err != nil {
					result, err = s.runError(ctx, err)
				}
			}

			s.setResult(result, err)
			if err != nil {
				return result, err
			}
		}
		return result, nil
	}

	if await {
		return runing()
	}

	go func() {
		_, err := runing()
		if err != nil {
			logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, s.StepId, err.Error())
		}
	}()

	result = et.Json{}
	return result, nil
}

/**
* runResilence
* @return (bool, error)
**/
func (s *Instance) runResilence(ctx et.Json, await bool, err error) (et.Json, error) {
	if s.flow.TotalAttempts == 0 {
		return et.Json{}, err
	}

	if s.resilience == nil {
		resilience, err := resilience.New()
		if err != nil {
			return et.Json{}, err
		}
		s.resilience = resilience
	}

	description := fmt.Sprintf("flow: %s,  %s", s.flow.Name, s.flow.Description)
	resilence, err := s.resilience.LoadInstance(resilience.Params{
		Id:            s.ID,
		Tag:           "workflow",
		Description:   description,
		TotalAttempts: s.flow.TotalAttempts,
		Interval:      s.flow.TimeAttempts,
		Tags:          s.Tags,
		Fn:            s.Run,
		FnArgs:        []interface{}{ctx, await},
	})
	if err != nil {
		return et.Json{}, err
	}

	res, err := resilence.Run()
	if err != nil {
		return et.Json{}, err
	}

	if len(res) == 0 {
		return et.Json{}, errors.New(MSG_RESILIENCE_NO_RESULT)
	}

	result, ok := res[0].(et.Json)
	if !ok {
		return et.Json{}, errors.New(MSG_RESILIENCE_NO_RESULT)
	}

	return result, nil
}

/**
* rollback
* @return et.Json, error
**/
func (s *Instance) runError(ctx et.Json, err error) (et.Json, error) {
	if s.Step == nil {
		return et.Json{}, err
	}

	connection, exists := s.flow.getError(s.Step.ID)
	if !exists {
		return et.Json{}, err
	}

	step, exists := s.flow.getStep(connection.Target.StepId)
	if !exists {
		return et.Json{}, err
	}

	step.OnStatus(func(status Status) {
		s.setStatus(status)
	})
	result, errStep := step.Run(s, ctx)
	if errStep != nil {
		return et.Json{}, errStep
	}

	return result, err
}
