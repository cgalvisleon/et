package jwf

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

const (
	MANUAL   string = "manual"
	WEBHOOK  string = "webhook"
	CRON     string = "cron"
	SCHEDULE string = "schedule"
)

var (
	ErrrFlowNotFound = errors.New(MSG_FLOW_NOT_FOUND)
)

type Port string

const (
	PortInput  Port = "input"
	PortOutput Port = "output"
	PortError  Port = "error"
)

type StepConnection struct {
	StepId string `json:"steper_id"`
	Port   Port   `json:"port"`
	Index  int    `json:"index"`
}

type Connection struct {
	ID     string          `json:"id"`
	Source *StepConnection `json:"source"`
	Target *StepConnection `json:"target"`
	Kind   Port            `json:"kind"`
}

type Node struct {
	ID      string `json:"id"`
	ErrorId string `json:"error_id"`
}

type Trigger struct {
	Tag     string `json:"tag"`
	StartId string `json:"start_id"`
}

type FlowDefinition struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TenantId    string    `json:"tenant_id"`
	ID          string    `json:"id"`
	Tag         string    `json:"tag"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
}

type Flow struct {
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	WorkflowId    string                   `json:"workflow_id"`
	ID            string                   `json:"id"`
	Tag           string                   `json:"tag"`
	Title         string                   `json:"title"`
	Description   string                   `json:"description"`
	Version       string                   `json:"version"`
	Steps         map[string]*Step         `json:"steps"`
	Connections   []*Connection            `json:"connections"`
	Triggers      []*Trigger               `json:"triggers"`
	TotalAttempts int                      `json:"total_attempts"`
	TimeAttempts  time.Duration            `json:"time_attempts"`
	TimeAwait     time.Duration            `json:"time_await"`
	Published     bool                     `json:"published"`
	AuditLog      []et.Json                `json:"audit_log"`
	isDebug       bool                     `json:"-"`
	isChanged     bool                     `json:"-"`
	workflow      *WorkFlow                `json:"-"`
	store         Store                    `json:"-"`
	onSave        []func(flow *Flow) error `json:"-"`
	onDelete      []func(flow *Flow) error `json:"-"`
	step          *Step                    `json:"-"`
	err           error                    `json:"-"`
}

/**
* newFlow
* @param tag, title, version, userId string
* @return *Flow
**/
func (s *WorkFlow) newFlow(tag, title, version, userId string) *Flow {
	if version == "" {
		version = "1.0.0"
	}

	now := timezone.Now()
	result := &Flow{
		CreatedAt:     now,
		UpdatedAt:     now,
		WorkflowId:    s.ID,
		ID:            reg.UUID(),
		Tag:           tag,
		Title:         title,
		Description:   "",
		Version:       version,
		Steps:         make(map[string]*Step),
		Connections:   make([]*Connection, 0),
		Triggers:      make([]*Trigger, 0),
		TotalAttempts: 0,
		TimeAttempts:  0,
		TimeAwait:     1 * time.Minute,
		Published:     false,
		AuditLog:      make([]et.Json, 0),
	}
	s.addAuditLog(userId, "new_flow")
	return result.up(s)
}

/**
* loadFlow
* @param id string
* @return *Flow, error
**/
func (s *WorkFlow) loadFlow(id string) (*Flow, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	result, exists := s.getFlow(id)
	if exists {
		return result, nil
	}

	exists, err := s.store.Get("flows", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrrFlowNotFound
	}

	return result.up(s), nil
}

/**
* up
* @param workflow *WorkFlow
* @return *Flow
**/
func (s *Flow) up(workflow *WorkFlow) *Flow {
	s.workflow = workflow
	s.store = workflow.store
	s.isDebug = workflow.isDebug
	s.onSave = make([]func(flow *Flow) error, 0)
	s.onDelete = make([]func(flow *Flow) error, 0)
	s.OnSave(func(flow *Flow) error {
		key := fmt.Sprintf("flow:%s", flow.ID)
		event.Publish(key, flow.ToJson())
		return nil
	})
	s.OnDelete(func(flow *Flow) error {
		key := fmt.Sprintf("flow:%s:delete", flow.ID)
		event.Publish(key, et.Json{
			"id": flow.ID,
		})
		return nil
	})
	workflow.addFlow(s)
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Flow) addAuditLog(userId string, action string) {
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
}

/**
* OnSave
* @param fn func(flow *Flow) error
* @return *Jrex
**/
func (s *Flow) OnSave(fn func(flow *Flow) error) *Flow {
	if s.onSave == nil {
		s.onSave = make([]func(flow *Flow) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(flow *Flow) error
* @return *Step
**/
func (s *Flow) OnDelete(fn func(flow *Flow) error) *Flow {
	if s.onDelete == nil {
		s.onDelete = make([]func(flow *Flow) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Flow) Save() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false
	data := s.ToJson()

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	err := s.store.Set("flows", s.ID, s.WorkflowId, s)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s)
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
func (s *Flow) ToJson() et.Json {
	return et.Json{
		"created_at":     timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":     timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"workflow_id":    s.WorkflowId,
		"id":             s.ID,
		"tag":            s.Tag,
		"title":          s.Title,
		"description":    s.Description,
		"version":        s.Version,
		"steps":          s.Steps,
		"connections":    s.Connections,
		"total_attempts": s.TotalAttempts,
		"time_attempts":  s.TimeAttempts.String(),
		"published":      s.Published,
		"audit_log":      s.AuditLog,
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
* setTimeAwait
* @param time time.Duration, userId string
* @return *Flow
**/
func (s *Flow) setTimeAwait(time time.Duration, userId string) *Flow {
	s.TimeAwait = time
	s.addAuditLog(userId, "set_time_await")
	return s
}

/**
* getTrigger
* @param tag string
* @return *Trigger, error
**/
func (s *Flow) getTrigger(tag string) (*Trigger, bool) {
	idx := slices.IndexFunc(s.Triggers, func(trigger *Trigger) bool {
		return trigger.Tag == tag
	})

	if idx == -1 {
		return nil, false
	}

	return s.Triggers[idx], true
}

/**
* getTarget
* @param stepId string
* @return *Connection, error
**/
func (s *Flow) getTarget(stepId string, index int) (*Step, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Kind == PortOutput && connection.Source.StepId == stepId && connection.Target.Index == index
	})

	if idx == -1 {
		return nil, false
	}

	conn := s.Connections[idx]
	step, exists := s.getStep(conn.Target.StepId)
	if !exists {
		return nil, false
	}

	return step, true
}

/**
* getSource
* @param stepId string, index int
* @return *Connection, bool
**/
func (s *Flow) getSource(stepId string, index int) (*Step, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Kind == PortOutput && connection.Target.StepId == stepId && connection.Source.Index == index
	})

	if idx == -1 {
		return nil, false
	}

	conn := s.Connections[idx]
	step, exists := s.getStep(conn.Source.StepId)
	if !exists {
		return nil, false
	}

	return step, true
}

/**
* getError
* @param stepId string, index int
* @return *Connection, bool
**/
func (s *Flow) getError(stepId string, index int) (*Step, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Kind == PortError && connection.Source.StepId == stepId && connection.Target.Index == index
	})

	if idx == -1 {
		return nil, false
	}

	conn := s.Connections[idx]
	step, exists := s.getStep(conn.Source.StepId)
	if !exists {
		return nil, false
	}

	return step, true
}

/**
* getStep
* @param stepId string
* @return *Step, bool
**/
func (s *Flow) getStep(stepId string) (*Step, bool) {
	step, exists := s.Steps[stepId]
	if !exists {
		return nil, false
	}

	return step, true
}

/**
* addConnection
* @param sourceId string, targetId string, kind Port
* @return *Connection, bool
**/
func (s *Flow) addConnection(sourceId string, targetId string, index int, kind Port) (*Connection, bool) {
	idx := slices.IndexFunc(s.Connections, func(connection *Connection) bool {
		return connection.Kind == kind && connection.Source.StepId == sourceId && connection.Target.Index == index
	})

	if idx != -1 {
		return s.Connections[idx], false
	}

	_, exists := s.getStep(sourceId)
	if !exists {
		return nil, false
	}

	_, exists = s.getStep(targetId)
	if !exists {
		return nil, false
	}

	result := &Connection{
		ID: reg.UUID(),
		Source: &StepConnection{
			StepId: sourceId,
			Port:   PortOutput,
			Index:  0,
		},
		Target: &StepConnection{
			StepId: targetId,
			Port:   PortInput,
			Index:  index,
		},
		Kind: kind,
	}

	s.Connections = append(s.Connections, result)
	return result, true
}

/**
* addStep
* @param tag, version, title string, kind Port, fn func(instance *Instance, ctx et.Json) (et.Json, error), userId string
* @return *Flow
**/
func (s *Flow) addStep(kind Kind, tag, version, title string, port Port, fn func(instance *Instance, ctx et.Json) (et.Json, error), userId string) *Flow {
	result := s.workflow.newStep(kind, "", tag, version, title, userId)
	result.Definition = fn
	s.Steps[result.ID] = result

	if s.step == nil {
		s.step = result
		return s
	}

	_, exists := s.addConnection(s.step.ID, result.ID, 0, port)
	if !exists {
		s.err = errors.New(MSG_INVALID_SOURCE)
		return s
	}

	s.step = result

	return s
}

/**
* Step
* @param tag, version, title string, fn func(instance *Instance, ctx et.Json) (et.Json, error), userId string
* @return *Flow
**/
func (s *Flow) Step(tag, title string, fn func(instance *Instance, ctx et.Json) (et.Json, error)) *Flow {
	if len(s.Steps) == 0 {
		result := s.workflow.newStep(KindTrigger, "", tag, "1.0.0", title, s.ID)
		result.Definition = fn
		s.Steps[result.ID] = result
		s.step = result

		s.Triggers = append(s.Triggers, &Trigger{
			Tag:     tag,
			StartId: result.ID,
		})
		return s
	}

	return s.addStep(KindAction, tag, "1.0.0", title, PortOutput, fn, s.ID)
}

/**
* Error
* @param stepId string
* @return *Flow
**/
func (s *Flow) Error(tag, version, title string, fn func(instance *Instance, ctx et.Json) (et.Json, error)) *Flow {
	if len(s.Steps) == 0 {
		s.err = errors.New(MSG_INVALID_SOURCE)
		return s
	}

	return s.addStep(KindAction, tag, version, title, PortError, fn, s.ID)
}

/**
* IsError
* @return error
**/
func (s *Flow) IsError() error {
	return s.err
}

/**
* AddStep
* @param stepDef et.Json
* @return *Flow, error
**/
func (s *Flow) AddStep(stepDef et.Json, userId string) (*Flow, error) {
	var step *Step
	bt, err := stepDef.ToByte()
	if err != nil {
		return s, err
	}

	err = json.Unmarshal(bt, &step)
	if err != nil {
		return s, err
	}

	step.ID = reg.UUID()
	step.up(s.workflow)
	step.OwnerId = s.ID
	s.addAuditLog(userId, "add_step")
	s.Steps[step.ID] = step
	if step.Kind == KindTrigger {
		s.Triggers = append(s.Triggers, &Trigger{
			Tag:     step.Tag,
			StartId: step.ID,
		})
	}
	return s, nil
}

/**
* Publish
* @return []et.Json, error
**/
func (s *Flow) Publish() ([]et.Json, error) {
	s.Published = true
	s.addAuditLog(s.ID, "publish")
	results := make([]et.Json, 0)
	for _, step := range s.Steps {
		result, err := step.runOnPublish(s, et.Json{})
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
