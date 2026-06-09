package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Flow struct {
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	TenantId      string        `json:"tenant_id"`
	ProjectId     string        `json:"project_id"`
	ID            string        `json:"id"`
	Tag           string        `json:"tag"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Version       string        `json:"version"`
	WorkflowId    string        `json:"workflow_id"`
	Steps         []*Step       `json:"steps"`
	Connections   []*Connection `json:"connections"`
	TotalAttempts int           `json:"total_attempts"`
	TimeAttempts  time.Duration `json:"time_attempts"`
	Public        bool          `json:"public"`
	AuditLog      []et.Json     `json:"audit_log"`
	UserID        string        `json:"-"`
	isDebug       bool          `json:"-"`
	isChanged     bool          `json:"-"`
	store         Store         `json:"-"`
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
	id := fmt.Sprintf("flow:%s:%s:%s", params.Tag, s.ID, params.Version)
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
		Steps:         make([]*Step, 0),
		Connections:   make([]*Connection, 0),
		TotalAttempts: 0,
		TimeAttempts:  0,
		Public:        false,
		AuditLog:      make([]et.Json, 0),
		UserID:        params.UserID,
		store:         s.store,
	}
	return result
}

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

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	s.isChanged = false

	err := s.store.Set("flow", s.ID, s.TenantId, s.ProjectId, s, s.UserID)
	if err != nil {
		return err
	}

	return nil
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
