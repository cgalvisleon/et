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

type Instance struct {
	StartedAt   time.Time            `json:"started_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DoneAt      time.Time            `json:"done_at"`
	TenantId    string               `json:"tenant_id"`
	ProjectId   string               `json:"project_id"`
	ID          string               `json:"id"`
	FlowId      string               `json:"flow_id"`
	Code        string               `json:"code"`
	Title       string               `json:"title"`
	Status      Status               `json:"status"`
	Ctx         et.Json              `json:"ctx"`
	Ctxs        map[int]et.Json      `json:"ctxs"`
	Results     map[int]*Result      `json:"results"`
	Params      et.Json              `json:"params"`
	Tags        et.Json              `json:"tags"`
	Traces      []et.Json            `json:"traces"`
	Steper      *Steper              `json:"steper"`
	CurrentStep int                  `json:"current_step"`
	IsDone      bool                 `json:"is_done"`
	IsStop      bool                 `json:"is_stop"`
	AuditLog    []et.Json            `json:"audit_log"`
	UserID      string               `json:"-"`
	isDebug     bool                 `json:"-"`
	store       Store                `json:"-"`
	flow        *Flow                `json:"-"`
	step        *Step                `json:"-"`
	resilience  *resilience.Instance `json:"-"`
}

type InstanceParams struct {
	TenantId  string `json:"tenant_id"`
	ProjectId string `json:"project_id"`
	FlowId    string `json:"flow_id"`
	Steper    string `json:"steper"`
	UserID    string `json:"user_id"`
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

	steper, exists := flow.Stepers[params.Steper]
	if !exists {
		return nil, errors.New(MSG_STEPER_NOT_FOUND)
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
	id := reg.GenULID("instance")
	result := &Instance{
		StartedAt:   now,
		TenantId:    params.TenantId,
		ProjectId:   params.ProjectId,
		ID:          id,
		FlowId:      params.FlowId,
		Code:        code,
		Title:       title,
		Status:      PENDING,
		Ctx:         et.Json{},
		Ctxs:        make(map[int]et.Json),
		Results:     make(map[int]*Result),
		Params:      et.Json{},
		Tags:        et.Json{},
		Traces:      make([]et.Json, 0),
		Steper:      steper,
		CurrentStep: -1,
		IsDone:      false,
		IsStop:      false,
		AuditLog:    make([]et.Json, 0),
		UserID:      params.UserID,
		store:       s.store,
		flow:        flow,
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

	flow, err := s.getFlow(result.FlowId, userId)
	if err != nil {
		return nil, err
	}

	result.store = s.store
	result.flow = flow
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
		"started_at":   timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at":   timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":      timezone.Format(s.DoneAt, timezone.RFC3339),
		"tenant_id":    s.TenantId,
		"id":           s.ID,
		"code":         s.Code,
		"title":        s.Title,
		"status":       s.Status,
		"ctx":          s.Ctx,
		"ctxs":         s.Ctxs,
		"results":      s.Results,
		"params":       s.Params,
		"tags":         s.Tags,
		"traces":       s.Traces,
		"steper":       s.Steper,
		"current_step": s.CurrentStep,
		"is_done":      s.IsDone,
		"is_stop":      s.IsStop,
		"audit_log":    s.AuditLog,
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
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.FlowId, s.Status, s.CurrentStep)
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
	s.Results[s.CurrentStep] = &Result{
		Step:   s.CurrentStep,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}
	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, s.CurrentStep, err.Error())
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
	s.Ctxs[s.CurrentStep] = ctx
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

	if index < 0 || index >= len(s.Steper.Connections) {
		return errors.New(MSG_INSTANCE_INVALID_STEP)
	}

	s.CurrentStep = index - 1
	return nil
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

	s.CurrentStep++
	if s.CurrentStep >= len(s.Steper.Connections) {
		s.setResult(et.Json{}, errors.New(MSG_INSTANCE_INVALID_CONNECTION))
		return false
	}

	connection := s.Steper.Connections[s.CurrentStep]
	if connection == nil {
		s.setResult(et.Json{}, errors.New(MSG_INSTANCE_INVALID_CONNECTION))
		return false
	}

	stepId := connection.Source.StepId
	step, exists := s.flow.Steps[stepId]
	if !exists {
		s.setResult(et.Json{}, errors.New(MSG_STEP_NOT_FOUND))
		return false
	}

	s.step = step
	return s.step != nil
}
