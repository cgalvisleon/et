package workflow

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

const (
	MANUAL   string = "manual"
	WEBHOOK  string = "webhook"
	CRON     string = "cron"
	SCHEDULE string = "schedule"
)

type Port string

const (
	PortInput  Port = "input"
	PortOutput Port = "output"
)

type StepConnection struct {
	StepId string `json:"steper_id"`
	Port   Port   `json:"port"`
	Index  int    `json:"index"`
}

type Connection struct {
	ID     string         `json:"id"`
	Tag    string         `json:"tag"`
	Source StepConnection `json:"source"`
	Target StepConnection `json:"target"`
	Kind   string         `json:"kind"`
}

type Trigger struct {
	Tag         string   `json:"tag"`
	Type        string   `json:"type"`
	StepId      string   `json:"step_id"`
	Connections []string `json:"connections"`
}

type Flow struct {
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	TenantId      string           `json:"tenant_id"`
	ProjectId     string           `json:"project_id"`
	ID            string           `json:"id"`
	Tag           string           `json:"tag"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	Version       string           `json:"version"`
	WorkflowId    string           `json:"workflow_id"`
	Steps         map[string]*Step `json:"steps"`
	Connections   []*Connection    `json:"connections"`
	Triggers      []*Trigger       `json:"triggers"`
	TotalAttempts int              `json:"total_attempts"`
	TimeAttempts  time.Duration    `json:"time_attempts"`
	Public        bool             `json:"public"`
	AuditLog      []et.Json        `json:"audit_log"`
	UserID        string           `json:"-"`
	isDebug       bool             `json:"-"`
	isChanged     bool             `json:"-"`
	store         Store            `json:"-"`
}

type FlowParams struct {
	TenantId    string `json:"tenant_id"`
	ProjectId   string `json:"project_id"`
	Tag         string `json:"tag"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
	UserID      string `json:"user_id"`
}

/**
* newFlow
* @param params FlowParams
* @return *Flow
**/
func (s *WorkFlow) newFlow(params FlowParams) *Flow {
	now := timezone.Now()
	if params.Version == "" {
		params.Version = "1.0.0"
	}
	params.Tag = utility.Normalize(params.Tag)
	id := fmt.Sprintf("flow:%s:%s", params.Tag, params.Version)
	result := &Flow{
		CreatedAt:     now,
		UpdatedAt:     now,
		TenantId:      params.TenantId,
		ProjectId:     params.ProjectId,
		ID:            id,
		Tag:           params.Tag,
		Title:         params.Title,
		Description:   params.Description,
		Version:       params.Version,
		WorkflowId:    s.ID,
		Steps:         make(map[string]*Step),
		Connections:   make([]*Connection, 0),
		Triggers:      make([]*Trigger, 0),
		TotalAttempts: 0,
		TimeAttempts:  0,
		Public:        false,
		AuditLog:      make([]et.Json, 0),
		UserID:        params.UserID,
		store:         s.store,
	}
	return result
}

/**
* loadFlow
* @param id, userId string
* @return *Flow, error
**/
func (s *WorkFlow) loadFlow(id, userId string) (*Flow, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	result := &Flow{}
	exists, err := s.store.GetByCollection("flow", id, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_FLOW_NOT_FOUND)
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
func (s *Flow) save() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    s.UserID,
		"action":     "save",
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	return s.store.Set("flow", s.ID, s.TenantId, s.WorkflowId, s, s.UserID)
}

/**
* delete
* @return error
**/
func (s *Flow) delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	return s.store.DeleteByCollection("flow", s.ID)
}

/**
* ToJson
* @return et.Json
**/
func (s *Flow) ToJson() et.Json {
	return et.Json{
		"created_at":     s.CreatedAt,
		"updated_at":     s.UpdatedAt,
		"tenant_id":      s.TenantId,
		"project_id":     s.ProjectId,
		"id":             s.ID,
		"tag":            s.Tag,
		"workflow_id":    s.WorkflowId,
		"title":          s.Title,
		"description":    s.Description,
		"version":        s.Version,
		"steps":          s.Steps,
		"connections":    s.Connections,
		"total_attempts": s.TotalAttempts,
		"time_attempts":  s.TimeAttempts.String(),
	}
}

/**
* ToString
* @return string
**/
func (s *Flow) ToString() string {
	return s.ToJson().ToString()
}

/**
* getTrigger
* @param tag string
* @return *Trigger, error
**/
func (s *Flow) getTrigger(tag string) (*Trigger, error) {
	idx := slices.IndexFunc(s.Triggers, func(trigger *Trigger) bool {
		return trigger.Tag == tag
	})

	if idx == -1 {
		return nil, errors.New(MSG_INSTANCE_TRIGGER_NOT_FOUND)
	}

	result := s.Triggers[idx]
	if result == nil {
		return nil, errors.New(MSG_INSTANCE_INVALID_TRIGGER)
	}

	return result, nil
}

/**
* getCurrentSource
* @param stepId string, index int
* @return *Connection, error
**/
func (s *Flow) getConnection(stepId string, index int) (*Connection, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Source.StepId == stepId && connection.Source.Index == index && connection.Kind == "output"
	})

	if idx == -1 {
		return nil, false
	}

	return s.Connections[idx], true
}

/**
* getCurrentError
* @param stepId string, index int
* @return *Connection
**/
func (s *Flow) getConnectionError(stepId string, index int) (*Connection, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Source.StepId == stepId && connection.Source.Index == index && connection.Kind == "error"
	})

	if idx == -1 {
		return nil, false
	}

	return s.Connections[idx], true
}

/**
* getCurrent
* @param stepId string, index int
* @return *Connection, error
**/
func (s *Flow) getCurrent(stepId string, index int) (*Current, bool) {
	conn, exists := s.getConnection(stepId, index)
	if !exists {
		return nil, false
	}

	source := s.getStep(conn.Source.StepId)
	target := s.getStep(conn.Target.StepId)

	result := &Current{
		Source: source,
		Target: target,
	}

	connError, exists := s.getConnectionError(stepId, index)
	if exists {
		result.Error = s.getStep(connError.Target.StepId)
	}

	return result, true
}

/**
* getStep
* @param stepId string
* @return *Step, bool
**/
func (s *Flow) getStep(stepId string) *Step {
	step, exists := s.Steps[stepId]
	if !exists {
		return nil
	}

	return step
}
