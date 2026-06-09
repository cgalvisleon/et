package workflow

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
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
)

var FlowStatusList map[Status]bool = map[Status]bool{
	PENDING:  true,
	RUNNING:  true,
	ROLLBACK: true,
	DONE:     true,
	FAILED:   true,
	CANCEL:   true,
}

type Result struct {
	Step   int     `json:"step"`
	Ctx    et.Json `json:"ctx"`
	Result et.Json `json:"result"`
	Error  string  `json:"error"`
}

type Current struct {
	Index      int         `json:"index"`
	Step       *Step       `json:"step"`
	Connection *Connection `json:"connection"`
}

type Instance struct {
	StartedAt  time.Time            `json:"started_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	DoneAt     time.Time            `json:"done_at"`
	TenantId   string               `json:"tenant_id"`
	ProjectId  string               `json:"project_id"`
	ID         string               `json:"id"`
	Flow       *Flow                `json:"flow"`
	Code       string               `json:"code"`
	Title      string               `json:"title"`
	Status     Status               `json:"status"`
	Ctx        et.Json              `json:"ctx"`
	Ctxs       map[int]et.Json      `json:"ctxs"`
	Results    map[int]*Result      `json:"results"`
	Params     et.Json              `json:"params"`
	Tags       et.Json              `json:"tags"`
	Traces     []et.Json            `json:"traces"`
	Trigger    Trigger              `json:"trigger"`
	Current    *Current             `json:"current"`
	IsDone     bool                 `json:"is_done"`
	IsStop     bool                 `json:"is_stop"`
	AuditLog   []et.Json            `json:"audit_log"`
	UserID     string               `json:"-"`
	isDebug    bool                 `json:"-"`
	store      Store                `json:"-"`
	resilience *resilience.Instance `json:"-"`
}

type InstanceParams struct {
	TenantId  string  `json:"tenant_id"`
	ProjectId string  `json:"project_id"`
	Tag       string  `json:"tag"`
	Trigger   Trigger `json:"trigger"`
	UserID    string  `json:"user_id"`
}

/**
* newInstance
* @param params InstanceParams
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(params InstanceParams) (*Instance, error) {
	flow, exists := s.Flows[params.Tag]
	if !exists {
		return nil, errors.New(MSG_FLOW_NOT_FOUND)
	}

	code := ""
	if s.store == nil {
		var err error
		code, err = s.store.GetCode(params.Tag)
		if err != nil {
			return nil, err
		}
	}

	title := flow.Title
	if code != "" {
		title = fmt.Sprintf("%s %s", flow.Title, code)
	}

	now := timezone.Now()
	params.Tag = utility.Normalize(params.Tag)
	id := reg.GenULID("instance")
	result := &Instance{
		StartedAt: now,
		TenantId:  params.TenantId,
		ProjectId: params.ProjectId,
		ID:        id,
		Flow:      flow,
		Code:      code,
		Title:     title,
		Status:    PENDING,
		Ctx:       et.Json{},
		Ctxs:      make(map[int]et.Json),
		Results:   make(map[int]*Result),
		Params:    et.Json{},
		Tags:      et.Json{},
		Traces:    make([]et.Json, 0),
		Trigger:   params.Trigger,
		Current:   &Current{Index: -1, Step: nil, Connection: nil},
		IsDone:    false,
		IsStop:    false,
		AuditLog:  make([]et.Json, 0),
		UserID:    params.UserID,
		store:     s.store,
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
	exists, err := s.store.Get("instance", id, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_INSTANCE_NOT_FOUND)
	}

	result.store = s.store
	result.isDebug = s.isDebug
	result.UserID = userId
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

	err := s.store.Set("instance", s.ID, s.TenantId, s.ProjectId, s, s.UserID)
	if err != nil {
		return err
	}

	return nil
}

/**
* delete
* @return error
**/
func (s *Instance) delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	err := s.store.Delete("instance", s.ID)
	if err != nil {
		return err
	}

	return nil
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
		"tenant_id":  s.TenantId,
		"id":         s.ID,
		"flow":       s.Flow.ToJson(),
		"code":       s.Code,
		"title":      s.Title,
		"status":     s.Status,
		"ctx":        s.Ctx,
		"ctxs":       s.Ctxs,
		"results":    s.Results,
		"params":     s.Params,
		"tags":       s.Tags,
		"traces":     s.Traces,
		"trigger":    s.Trigger,
		"current":    s.Current,
		"is_done":    s.IsDone,
		"is_stop":    s.IsStop,
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
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.Flow.Tag, s.Status, s.Current.Index)
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
	s.Results[s.Current.Index] = &Result{
		Step:   s.Current.Index,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}
	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.Flow.Tag, s.Current.Index, err.Error())
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
	s.Ctxs[s.Current.Index] = s.Ctx.Clone()
	return s.Ctx
}

/**
* setCurrent
* @param step int
**/
func (s *Instance) setCurrentStep(index int) error {
	if s.IsDone {
		return errors.New(MSG_INSTANCE_ALREADY_DONE)
	}

	steps, exists := s.Flow.Connections[s.Trigger]
	if !exists {
		return errors.New(MSG_INSTANCE_TRIGGER_NOT_FOUND)
	}

	if index < 0 || index >= len(steps) {
		return errors.New(MSG_INSTANCE_INVALID_STEP)
	}

	s.Current.Index = index
	s.Current.Connection = steps[index]
	if s.Current.Connection == nil {
		return errors.New(MSG_INSTANCE_CONNECTION_NOT_FOUND)
	}

	step, exists := s.Flow.Steps[s.Current.Connection.Source.StepId]
	if !exists {
		return errors.New(MSG_STEP_NOT_FOUND)
	}
	s.CurrentStep = step
	return nil
}
