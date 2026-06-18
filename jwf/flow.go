package jwf

import (
	"errors"
	"slices"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
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

type Flow struct {
	CreatedAt     time.Time                               `json:"created_at"`
	UpdatedAt     time.Time                               `json:"updated_at"`
	TenantId      string                                  `json:"tenant_id"`
	ID            string                                  `json:"id"`
	Tag           string                                  `json:"tag"`
	Title         string                                  `json:"title"`
	Description   string                                  `json:"description"`
	Version       string                                  `json:"version"`
	WorkflowId    string                                  `json:"workflow_id"`
	Steps         map[string]*Step                        `json:"steps"`
	Connections   []*Connection                           `json:"connections"`
	Triggers      []*Trigger                              `json:"triggers"`
	TotalAttempts int                                     `json:"total_attempts"`
	TimeAttempts  time.Duration                           `json:"time_attempts"`
	Public        bool                                    `json:"public"`
	AuditLog      []et.Json                               `json:"audit_log"`
	isDebug       bool                                    `json:"-"`
	isChanged     bool                                    `json:"-"`
	workflow      *WorkFlow                               `json:"-"`
	store         Store                                   `json:"-"`
	onSave        []func(flow *Flow, userId string) error `json:"-"`
	onDelete      []func(flow *Flow, userId string) error `json:"-"`
	step          *Step                                   `json:"-"`
	err           error                                   `json:"-"`
}

/**
* newFlow
* @param tag, title, version string
* @return *Flow
**/
func (s *WorkFlow) newFlow(tag, title, version string) *Flow {
	if version == "" {
		version = "1.0.0"
	}
	id := reg.ULID()
	now := timezone.Now()
	result := &Flow{
		CreatedAt:     now,
		UpdatedAt:     now,
		TenantId:      s.TenantId,
		ID:            id,
		Tag:           tag,
		Title:         title,
		Description:   "",
		Version:       version,
		WorkflowId:    s.ID,
		Steps:         make(map[string]*Step),
		Connections:   make([]*Connection, 0),
		Triggers:      make([]*Trigger, 0),
		TotalAttempts: 0,
		TimeAttempts:  0,
		Public:        false,
		AuditLog:      make([]et.Json, 0),
		isDebug:       s.isDebug,
		workflow:      s,
		store:         s.store,
		onSave:        make([]func(flow *Flow, userId string) error, 0),
		onDelete:      make([]func(flow *Flow, userId string) error, 0),
	}
	return result
}

/**
* OnSave
* @param fn func(flow *Flow, userId string) error
* @return *Jrex
**/
func (s *Flow) OnSave(fn func(flow *Flow, userId string) error) *Flow {
	if s.onSave == nil {
		s.onSave = make([]func(flow *Flow, userId string) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(flow *Flow, userId string) error
* @return *Step
**/
func (s *Flow) OnDelete(fn func(flow *Flow, userId string) error) *Flow {
	if s.onDelete == nil {
		s.onDelete = make([]func(flow *Flow, userId string) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* getFlow
* @param id string
* @return *Flow, error
**/
func (s *WorkFlow) getFlow(id string) (*Flow, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	var result *Flow
	exists, err := s.store.Get("flow", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrrFlowNotFound
	}

	result.workflow = s
	result.store = s.store
	result.isDebug = s.isDebug
	result.onSave = make([]func(flow *Flow, userId string) error, 0)
	result.onDelete = make([]func(flow *Flow, userId string) error, 0)
	return result, nil
}

/**
* deleteFlow
* @return error
**/
func (s *WorkFlow) deleteFlow(id, userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	flow, err := s.getFlow(id)
	if err != nil {
		return err
	}

	err = s.store.Delete("flow", id)
	if err != nil {
		return err
	}

	for _, onDelete := range flow.onDelete {
		err := onDelete(flow, userId)
		if err != nil {
			return err
		}
	}

	return nil
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
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* save
* @return error
**/
func (s *Flow) save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false
	data := s.ToJson()

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	err := s.store.Set("flow", s.ID, s.TenantId, s.WorkflowId, s, userId)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s, userId)
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
		"tenant_id":      s.TenantId,
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
		"public":         s.Public,
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

/**
* IsError
* @return error
**/
func (s *Flow) IsError() error {
	return s.err
}

/**
* addStep
* @param tag, version, title string, kind Port, fn func(instance *Instance, ctx et.Json) (et.Json, error)
* @return *Flow
**/
func (s *Flow) addStep(tag, version, title string, kind Port, fn func(instance *Instance, ctx et.Json) (et.Json, error)) *Flow {
	step, err := s.workflow.newStep(KindFunction, tag, version, title)
	if err != nil {
		s.err = err
		return s
	}
	step.Definition = fn
	s.Steps[step.ID] = step

	if s.step == nil {
		s.err = errors.New(MSG_INVALID_SOURCE)
		return s
	}

	s.Connections = append(s.Connections, &Connection{
		ID: reg.ULID(),
		Source: &StepConnection{
			StepId: s.step.ID,
			Port:   PortOutput,
			Index:  0,
		},
		Target: &StepConnection{
			StepId: step.ID,
			Port:   PortInput,
			Index:  0,
		},
		Kind: kind,
	})

	if kind != PortError {
		s.step = step
	}

	return s
}

/**
* Step
* @param tag, version, title string, fn func(instance *Instance, ctx et.Json) (et.Json, error)
* @return *Flow
**/
func (s *Flow) Step(tag, version, title string, fn func(instance *Instance, ctx et.Json) (et.Json, error)) *Flow {
	if len(s.Steps) == 0 {
		step, err := s.workflow.newStep(KindTrigger, tag, version, title)
		if err != nil {
			s.err = err
			return s
		}
		step.Definition = fn
		s.Steps[step.ID] = step
		s.Connections = make([]*Connection, 0)
		s.Triggers = append(s.Triggers, &Trigger{
			Tag:     tag,
			StartId: step.ID,
		})
		return s
	}

	return s.addStep(tag, version, title, PortInput, fn)
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

	return s.addStep(tag, version, title, PortError, fn)
}
