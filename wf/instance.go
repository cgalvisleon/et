package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
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

type Instance struct {
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DoneAt      time.Time            `json:"done_at"`
	TenantId    string               `json:"tenant_id"`
	ProjectId   string               `json:"project_id"`
	ID          string               `json:"id"`
	Flow        *Flow                `json:"flow"`
	Code        string               `json:"code"`
	Title       string               `json:"title"`
	Status      Status               `json:"status"`
	Ctx         et.Json              `json:"ctx"`
	Ctxs        map[int]et.Json      `json:"ctxs"`
	Results     map[int]*Result      `json:"results"`
	Params      et.Json              `json:"params"`
	Tags        et.Json              `json:"tags"`
	Traces      []et.Json            `json:"traces"`
	CurrentStep int                  `json:"current_step"`
	IsDone      bool                 `json:"is_done"`
	IsStop      bool                 `json:"is_stop"`
	AuditLog    []et.Json            `json:"audit_log"`
	UserID      string               `json:"-"`
	isDebug     bool                 `json:"-"`
	isChanged   bool                 `json:"-"`
	store       Store                `json:"-"`
	workflow    *WorkFlow            `json:"-"`
	resilience  *resilience.Instance `json:"-"`
	jrex        *jrex.Jrex           `json:"-"`
}

type InstanceParams struct {
	TenantId  string `json:"tenant_id"`
	ProjectId string `json:"project_id"`
	Tag       string `json:"tag"`
	UserID    string `json:"user_id"`
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
		CreatedAt:   now,
		UpdatedAt:   now,
		TenantId:    params.TenantId,
		ProjectId:   params.ProjectId,
		ID:          id,
		Flow:        flow,
		Code:        code,
		Title:       title,
		Status:      PENDING,
		Ctx:         et.Json{},
		Ctxs:        make(map[int]et.Json),
		Results:     make(map[int]*Result),
		Params:      et.Json{},
		Tags:        et.Json{},
		Traces:      make([]et.Json, 0),
		CurrentStep: -1,
		IsDone:      false,
		IsStop:      false,
		AuditLog:    make([]et.Json, 0),
		UserID:      params.UserID,
		store:       s.store,
		workflow:    s,
		resilience:  nil,
		jrex:        nil,
		isDebug:     s.isDebug,
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

	if s.store != nil {
		err := s.store.Set("instance", s.ID, s.TenantId, s.ProjectId, s, s.UserID)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	return et.Json{
		"created_at":   timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":   timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":      timezone.Format(s.DoneAt, timezone.RFC3339),
		"tenant_id":    s.TenantId,
		"id":           s.ID,
		"flow":         s.Flow.ToJson(),
		"code":         s.Code,
		"title":        s.Title,
		"status":       s.Status,
		"ctx":          s.Ctx,
		"ctxs":         s.Ctxs,
		"results":      s.Results,
		"params":       s.Params,
		"tags":         s.Tags,
		"traces":       s.Traces,
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

	s.Status = status
	s.UpdatedAt = timezone.Now()
	switch s.Status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	case FAILED:
		s.IsDone = false
	}

	return s.save()
}
