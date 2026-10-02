package jwf

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
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

type Result struct {
	StepId string  `json:"step_id"`
	Ctx    et.Json `json:"ctx"`
	Result et.Json `json:"result"`
	Error  string  `json:"error"`
}

type Owner struct {
	From jsql.From
	Id   string `json:"id"`
}

type Current struct {
	SourceId   string `json:"source_id"`
	TargetId   string `json:"target_id"`
	ErrorId    string `json:"error_id"`
	Index      int    `json:"index"`
	IsFinished bool   `json:"is_finished"`
}

type Instance struct {
	StartedAt  time.Time                  `json:"started_at"`
	UpdatedAt  time.Time                  `json:"updated_at"`
	DoneAt     time.Time                  `json:"done_at"`
	ID         string                     `json:"id"`
	FlowId     string                     `json:"flow_id"`
	FlowTag    string                     `json:"flow_tag"`
	Code       string                     `json:"code"`
	Name       string                     `json:"name"`
	Status     Status                     `json:"status"`
	Ctx        et.Json                    `json:"ctx"`
	Ctxs       map[string]et.Json         `json:"ctxs"`
	Params     et.Json                    `json:"params"`
	Results    map[string]*Result         `json:"results"`
	Owners     []*Owner                   `json:"owners"`
	Tags       et.Json                    `json:"tags"`
	Trigger    *Trigger                   `json:"trigger"`
	Current    *Connection                `json:"current"`
	Step       *Step                      `json:"step"`
	IsDone     bool                       `json:"is_done"`
	AuditLog   []et.Json                  `json:"audit_log"`
	stop       bool                       `json:"-"`
	isDebug    bool                       `json:"-"`
	isChanged  bool                       `json:"-"`
	flow       *Flow                      `json:"-"`
	bindings   map[string]interface{}     `json:"-"`
	resilience *resilience.Resilience     `json:"-"`
	onChange   []func(data et.Json) error `json:"-"`
	mu         sync.Mutex                 `json:"-"`
}

/**
* NewInstance
* @return *Instance
**/
func (s *Flow) NewInstance(id, code, name string, trigger *Trigger) *Instance {
	now := timezone.Now()
	id = reg.GetUUID(id)
	result := &Instance{
		StartedAt: now,
		ID:        id,
		FlowId:    s.ID,
		FlowTag:   s.Tag,
		Code:      code,
		Name:      name,
		Ctx:       et.Json{},
		Ctxs:      make(map[string]et.Json),
		Params:    et.Json{},
		Owners:    make([]*Owner, 0),
		Results:   make(map[string]*Result),
		Tags:      et.Json{},
		Trigger:   trigger,
		IsDone:    false,
		AuditLog:  make([]et.Json, 0),
		flow:      s,
		bindings:  make(map[string]interface{}),
		onChange:  make([]func(data et.Json) error, 0),
		mu:        sync.Mutex{},
	}
	result.up()
	result.setStatus(CREATED)
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

	return result.up(), nil
}

/**
* up
* @return *Instance
**/
func (s *Instance) up() *Instance {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	s.wrapper()
	return s
}

/**
* wrapper
* @param step *Step
**/
func (s *Instance) wrapper() {
	s.bindings["goTo"] = func(idx int) {
		s.goTo(idx)
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
	s.bindings["owner"] = map[string]interface{}{
		"add": func(from jsql.From, id string) {
			s.Owners = append(s.Owners, &Owner{
				From: from,
				Id:   id,
			})
		},
		"get": func(id string) *Owner {
			for _, owner := range s.Owners {
				if owner.Id == id {
					return owner
				}
			}
			return nil
		},
		"remove": func(id string) {
			for i, owner := range s.Owners {
				if owner.Id == id {
					s.Owners = append(s.Owners[:i], s.Owners[i+1:]...)
					break
				}
			}
		},
		"list": func() []*Owner {
			return s.Owners
		},
	}
}

/**
* addAuditLog
* @param userId string, action interface{}
**/
func (s *Instance) addAuditLog(userId string, action interface{}) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
	for _, fn := range s.onChange {
		err := fn(s.ToJson())
		if err != nil {
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
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	return et.Json{
		"started_at": timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":    timezone.Format(s.DoneAt, timezone.RFC3339),
		"id":         s.ID,
		"flow_id":    s.FlowId,
		"flow_tag":   s.FlowTag,
		"code":       s.Code,
		"name":       s.Name,
		"status":     s.Status,
		"ctx":        s.Ctx,
		"ctxs":       s.Ctxs,
		"params":     s.Params,
		"results":    s.Results,
		"owners":     s.Owners,
		"tags":       s.Tags,
		"trigger":    s.Trigger,
		"current":    s.Current,
		"step":       s.Step,
		"is_done":    s.IsDone,
		"audit_log":  s.AuditLog,
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
* push
* @return error
**/
func (s *Instance) push() {
	for _, fn := range s.onChange {
		err := fn(s.ToJson())
		if err != nil {
			return
		}
	}
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
	}

	s.push()
}

/**
* setTrace
* @param step int, result et.Json, err error
* @return error
**/
func (s *Instance) setTrace(stepId string, result et.Json, err error, userId string) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}

	s.addAuditLog(userId, et.Json{
		"action":  "set_trace",
		"step_id": stepId,
		"ctx":     s.Ctx,
		"result":  result,
		"error":   errMessage,
	})
	s.push()
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) *Instance {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	stepId := ""
	if s.Current != nil {
		stepId = s.Current.ID
	}

	if stepId == "" {
		return s
	}

	s.Results[stepId] = &Result{
		StepId: stepId,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}

	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, stepId, err.Error())
	} else {
		s.push()
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.FlowId, stepId, s.Status)
	}

	return s
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
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	maps.Copy(s.Ctx, ctx)
	if s.Current != nil {
		stepId := s.Current.ID
		s.Ctxs[stepId] = ctx
	}
	s.push()
	return s.Ctx
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
* setBySource
* @param step *Step
* @return error
**/
func (s *Instance) setBySource(connection *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Current = connection
	s.Step, _ = s.flow.getStep(connection.Source.StepId)
	go s.push()
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
	stop, err := cache.Get(key, "")
	if err != nil {
		return false
	}

	if stop == "true" {
		return true
	}

	return false
}

/**
* goTo
* @param idx int
* @return bool
**/
func (s *Instance) goTo(idx int) bool {
	if s.isStop() {
		return false
	}

	if s.IsDone {
		return false
	}

	status := s.getStatus()
	if status == CANCEL {
		return false
	}

	if s.Current == nil {
		step, exists := s.flow.getOutput(s.Trigger.StartId, idx)
		if !exists {
			return false
		}
		s.setBySource(step)
	} else {
		step, exists := s.flow.getOutput(s.Current.Target.StepId, idx)
		if !exists {
			return false
		}
		s.setBySource(step)
	}

	return s.Step != nil
}

/**
* next
* @return bool
**/
func (s *Instance) next() bool {
	return s.goTo(0)
}

/**
* run
* @param ctx, tags et.Json, await bool, userId string
* @return et.Json, error
**/
func (s *Instance) Run(ctx et.Json, await bool, userId string) (et.Json, error) {
	var err error
	defer func() {
		if s.Current != nil {
			s.setTrace(s.Current.ID, ctx, err, userId)
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
		var result et.Json
		for s.next() {
			step := s.Step
			if step == nil {
				return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
			}

			ctx = s.setCtx(ctx)
			result, err = step.run(s, ctx)
			if err != nil {
				result, err = s.runResilence(ctx, err, userId)
				if err != nil {
					result, err = s.runError(ctx, err)
				}
			}

			s.setResult(result, err)
			if err != nil {
				return result, err
			}

			if s.IsDone {
				return result, nil
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
			logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, s.Current.ID, err.Error())
		}
	}()

	return et.Json{
		"instance_id": s.ID,
		"flow_id":     s.FlowId,
		"flow_tag":    s.FlowTag,
		"code":        s.Code,
		"name":        s.Name,
		"status":      "running",
	}, nil
}

/**
* runResilence
* @return (bool, error)
**/
func (s *Instance) runResilence(ctx et.Json, err error, userId string) (et.Json, error) {
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
		FnArgs:        []interface{}{ctx, userId},
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

	result, errStep := step.run(s, ctx)
	if errStep != nil {
		return et.Json{}, errStep
	}

	return result, err
}
