package workflow

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

const (
	SYSTEM   Status = "system"
	ACTIVE   Status = "active"
	ARCHIVED Status = "archived"
	PENDING  Status = "pending"
	RUNNING  Status = "running"
	ROLLBACK Status = "rollback"
	DONE     Status = "done"
	FAILED   Status = "failed"
	CANCEL   Status = "cancel"
	STOP     Status = "stop"
)

var (
	ErrorInstanceNotFound                 = errors.New(MSG_INSTANCE_NOT_FOUND)
	FlowStatusList        map[Status]bool = map[Status]bool{
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

type Current struct {
	Source     *Step `json:"source"`
	Target     *Step `json:"target"`
	Error      *Step `json:"error"`
	IsFinished bool  `json:"is_finished"`
}

type Instance struct {
	StartedAt    time.Time              `json:"started_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	DoneAt       time.Time              `json:"done_at"`
	TenantId     string                 `json:"tenant_id"`
	ProjectId    string                 `json:"project_id"`
	ID           string                 `json:"id"`
	FlowId       string                 `json:"flow_id"`
	Code         string                 `json:"code"`
	Title        string                 `json:"title"`
	Status       Status                 `json:"status"`
	Ctx          et.Json                `json:"ctx"`
	Ctxs         map[string]et.Json     `json:"ctxs"`
	Results      map[string]*Result     `json:"results"`
	Params       et.Json                `json:"params"`
	Tags         et.Json                `json:"tags"`
	Traces       []et.Json              `json:"traces"`
	TriggerTag   string                 `json:"trigger_tag"`
	Trigger      *Trigger               `json:"trigger"`
	Current      *Current               `json:"current"`
	CurrentIndex int                    `json:"current_index"`
	IsDone       bool                   `json:"is_done"`
	IsStop       bool                   `json:"is_stop"`
	AuditLog     []et.Json              `json:"audit_log"`
	Rollbacks    bool                   `json:"rollbacks"`
	UserID       string                 `json:"-"`
	isDebug      bool                   `json:"-"`
	store        Store                  `json:"-"`
	flow         *Flow                  `json:"-"`
	bindings     map[string]any         `json:"-"`
	resilience   *resilience.Resilience `json:"-"`
}

type InstanceParams struct {
	TenantId   string `json:"tenant_id"`
	ProjectId  string `json:"project_id"`
	ID         string `json:"id"`
	FlowId     string `json:"flow_id"`
	TriggerTag string `json:"trigger_tag"`
	UserID     string `json:"user_id"`
}

/**
* newInstance
* @param params InstanceParams
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(params InstanceParams) (*Instance, error) {
	flow, err := s.getFlow(params.FlowId, params.UserID)
	if err != nil {
		return nil, err
	}

	trigger, err := flow.getTrigger(params.TriggerTag)
	if err != nil {
		return nil, err
	}

	code := ""
	if s.store == nil {
		var err error
		code, err = s.store.GetCode(flow.Tag)
		if err != nil {
			return nil, err
		}
	}

	title := flow.Title
	if code != "" {
		title = fmt.Sprintf("%s %s", flow.Title, code)
	}

	now := timezone.Now()
	id := reg.TagULID("instance", params.ID)
	result := &Instance{
		StartedAt:  now,
		TenantId:   params.TenantId,
		ProjectId:  params.ProjectId,
		ID:         id,
		FlowId:     params.FlowId,
		Code:       code,
		Title:      title,
		Status:     PENDING,
		Ctx:        et.Json{},
		Ctxs:       make(map[string]et.Json),
		Results:    make(map[string]*Result),
		Params:     et.Json{},
		Tags:       et.Json{},
		Traces:     make([]et.Json, 0),
		TriggerTag: params.TriggerTag,
		Trigger:    trigger,
		IsDone:     false,
		IsStop:     false,
		AuditLog:   make([]et.Json, 0),
		UserID:     params.UserID,
		store:      s.store,
		flow:       flow,
		bindings:   make(map[string]any),
	}
	for k, v := range s.bindings {
		result.bindings[k] = v
	}
	return result, nil
}

/**
* loadInstance
* @param id, userId string
* @return *Instance, error
**/
func (s *WorkFlow) loadInstance(id, userId string) (*Instance, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	result := &Instance{}
	exists, err := s.store.GetByCollection("instance", id, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrorInstanceNotFound
	}

	flow, err := s.getFlow(result.FlowId, userId)
	if err != nil {
		return nil, err
	}

	trigger, err := flow.getTrigger(result.TriggerTag)
	if err != nil {
		return nil, err
	}

	result.store = s.store
	result.flow = flow
	result.Trigger = trigger
	result.isDebug = s.isDebug
	result.UserID = userId
	result.bindings = make(map[string]any)
	for k, v := range s.bindings {
		result.bindings[k] = v
	}
	return result, nil
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    s.UserID,
		"action":     "save",
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	key := fmt.Sprintf("%s:status", s.ID)
	cache.SetObject(key, s.Status, 1*time.Minute)
	return s.store.Set("instance", s.ID, s.TenantId, s.ProjectId, s, s.UserID)
}

/**
* delete
* @return error
**/
func (s *Instance) delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	key := fmt.Sprintf("%s:status", s.ID)
	cache.Delete(key)
	return s.store.DeleteByCollection("instance", s.ID)
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	return et.Json{
		"started_at":  timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":     timezone.Format(s.DoneAt, timezone.RFC3339),
		"tenant_id":   s.TenantId,
		"id":          s.ID,
		"code":        s.Code,
		"title":       s.Title,
		"status":      s.Status,
		"ctx":         s.Ctx,
		"ctxs":        s.Ctxs,
		"results":     s.Results,
		"params":      s.Params,
		"tags":        s.Tags,
		"traces":      s.Traces,
		"trigger_tag": s.TriggerTag,
		"trigger":     s.Trigger,
		"current":     s.Current,
		"is_done":     s.IsDone,
		"is_stop":     s.IsStop,
		"audit_log":   s.AuditLog,
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
* setStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	if s.Status == status {
		return nil
	}

	s.UpdatedAt = timezone.Now()
	s.Status = status
	switch s.Status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	}

	if status != FAILED {
		step := ""
		if s.Current != nil && s.Current.Source != nil {
			step = s.Current.Source.Title
		}
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.FlowId, s.Status, step)
	}

	return s.save()
}

/**
* setTag
* @param tags et.Json
* @return et.Json
**/
func (s *Instance) setTag(tags et.Json) et.Json {
	maps.Copy(s.Tags, tags)
	return s.Tags
}

/**
* setParams
* @param params et.Json
* @return et.Json
**/
func (s *Instance) setParams(params et.Json) et.Json {
	maps.Copy(s.Params, params)
	return s.Params
}

/**
* setTrace
* @param step int, result et.Json, err error
* @return error
**/
func (s *Instance) setTrace(stepId string, result et.Json, err error) error {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	now := timezone.Now()
	s.Traces = append(s.Traces, et.Json{
		"created_at": now,
		"step_id":    stepId,
		"ctx":        s.Ctx,
		"result":     result,
		"error":      errMessage,
	})
	return s.save()
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) (et.Json, error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	stepId := ""
	if s.Current != nil && s.Current.Source != nil {
		stepId = s.Current.Source.ID
	}

	if stepId == "" {
		return result, err
	}

	s.Results[stepId] = &Result{
		StepId: stepId,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}
	if err != nil {
		s.setStatus(FAILED)
		step := ""
		if s.Current != nil && s.Current.Source != nil {
			step = s.Current.Source.Title
		}
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, step, err.Error())
	}
	return result, err
}

/**
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	maps.Copy(s.Ctx, ctx)
	stepId := ""
	if s.Current != nil && s.Current.Source != nil {
		stepId = s.Current.Source.ID
		s.Ctxs[stepId] = ctx
	}
	return s.Ctx
}

/**
* next
* @return bool
**/
func (s *Instance) next() bool {
	if s.IsStop {
		return false
	}

	if s.IsDone {
		return false
	}

	key := fmt.Sprintf("%s:status", s.ID)
	status, err := cache.Get(key, string(s.Status))
	if err != nil {
		return false
	}

	if status == string(CANCEL) {
		return false
	} else if status == string(STOP) {
		return false
	}

	if s.Current == nil {
		current, exists := s.flow.getCurrent(s.Trigger.StepId, s.CurrentIndex)
		if !exists {
			return false
		}
		s.Current = current
	} else {
		target := s.Current.Target
		if s.Rollbacks {
			target = s.Current.Error
		}

		if target == nil {
			return false
		}

		current, exists := s.flow.getCurrent(target.ID, s.CurrentIndex)
		if !exists {
			return false
		}
		s.Current = current
	}

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
		s.setTrace(s.Current.Source.ID, ctx, err)
	}()

	if s.Status == DONE {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
		return et.Json{}, err
	} else if s.Status == RUNNING {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
		return et.Json{}, err
	} else if s.Status == ROLLBACK {
		err = fmt.Errorf(MSG_INSTANCE_ROLLBACK, s.ID)
		return et.Json{}, err
	} else if s.Status == CANCEL {
		err = fmt.Errorf(MSG_INSTANCE_CANCEL, s.ID)
		return et.Json{}, err
	}

	var result et.Json
	for s.next() {
		step := s.Current.Source
		if step == nil {
			return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
		}

		ctx = s.setCtx(ctx)
		result, err = step.run(s, ctx)
		if err != nil {
			result, err := s.rollback(result, err)
			if err != nil {
				return result, err
			}
			continue
		}

		s.setResult(result, err)

		if s.IsDone {
			return result, nil
		}

		if s.IsStop || step.Stop {
			return result, nil
		}
	}

	return result, err
}

/**
* rollback
* @return et.Json, error
**/
func (s *Instance) rollback(result et.Json, err error) (et.Json, error) {
	if s.flow.TotalAttempts > 0 {
		result, err := s.startResilence()
		if err == nil {
			return result, nil
		}
	}

	if s.Rollbacks {
		return result, err
	}
	s.Rollbacks = true
	s.setResult(result, err)

	return result, err
}

/**
* startResilence
* @return (bool, error)
**/
func (s *Instance) startResilence() (et.Json, error) {
	if s.resilience == nil {
		resilience, err := resilience.New(s.store)
		if err != nil {
			return et.Json{}, err
		}
		s.resilience = resilience
	}

	description := fmt.Sprintf("flow: %s,  %s", s.flow.Title, s.flow.Description)
	resilence := s.resilience.LoadInstance(resilience.Params{
		TenantId:      s.TenantId,
		Id:            s.ID,
		Tag:           "instance",
		Description:   description,
		OwnerId:       s.ProjectId,
		TotalAttempts: s.flow.TotalAttempts,
		Interval:      s.flow.TimeAttempts,
		Tags:          s.Tags,
		UserId:        s.UserID,
		Fn:            s.run,
		FnArgs:        []interface{}{s.Ctx},
	})
	res, err := resilence.Run(s.UserID)
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
