package jwf

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
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
	ErrrFlowNotFound    = errors.New(MSG_FLOW_NOT_FOUND)
	ErrrTriggerNotFound = errors.New(MSG_TRIGGER_NOT_FOUND)
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

type fnStep func(instance *Instance, ctx et.Json) (et.Json, error)

type Flow struct {
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
	OwnerId       string                     `json:"owner_id"`
	ID            string                     `json:"id"`
	Tag           string                     `json:"tag"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	Version       string                     `json:"version"`
	Steps         map[string]*Step           `json:"steps"`
	Connections   []*Connection              `json:"connections"`
	Triggers      []*Trigger                 `json:"triggers"`
	TotalAttempts int                        `json:"total_attempts"`
	TimeAttempts  time.Duration              `json:"time_attempts"`
	TimeAwait     time.Duration              `json:"time_await"`
	Resources     []et.Json                  `json:"resources"`
	Published     bool                       `json:"published"`
	AuditLog      []et.Json                  `json:"audit_log"`
	isDebug       bool                       `json:"-"`
	isChanged     bool                       `json:"-"`
	onChange      []func(data et.Json) error `json:"-"`
	step          *Step                      `json:"-"`
}

/**
* newFlow
* @param tag, name, version, ownerId, userId string
* @return *Flow
**/
func NewFlow(tag, name, version, ownerId, userId string) *Flow {
	if version == "" {
		version = "1.0.0"
	}

	now := timezone.Now()
	result := &Flow{
		CreatedAt:     now,
		UpdatedAt:     now,
		OwnerId:       ownerId,
		ID:            reg.UUID(),
		Tag:           tag,
		Name:          name,
		Description:   "",
		Version:       version,
		Steps:         make(map[string]*Step),
		Connections:   make([]*Connection, 0),
		Triggers:      make([]*Trigger, 0),
		TotalAttempts: 0,
		TimeAttempts:  0,
		TimeAwait:     10 * time.Minute,
		Resources:     make([]et.Json, 0),
		Published:     false,
		AuditLog:      make([]et.Json, 0),
	}
	result.up()
	result.addAuditLog(userId, "new_flow")
	return result
}

/**
* LoadFlow
* @param def et.Json
* @return *Flow, error
**/
func LoadFlow(def et.Json) (*Flow, error) {
	bt, err := def.ToByte()
	if err != nil {
		return nil, err
	}

	var result *Flow
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	return result.up(), nil
}

/**
* up
* @param workflow *WorkFlow
* @return *Flow
**/
func (s *Flow) up() *Flow {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
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
	for _, fn := range s.onChange {
		err := fn(s.ToJson())
		if err != nil {
			return
		}
	}
}

/**
* OnChange
* @param fn func(data et.Json) error
* @return *Flow
**/
func (s *Flow) OnChange(fn func(data et.Json) error) *Flow {
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
func (s *Flow) ToJson() et.Json {
	return et.Json{
		"created_at":     timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":     timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"owner_id":       s.OwnerId,
		"id":             s.ID,
		"tag":            s.Tag,
		"name":           s.Name,
		"description":    s.Description,
		"version":        s.Version,
		"steps":          s.Steps,
		"connections":    s.Connections,
		"triggers":       s.Triggers,
		"total_attempts": s.TotalAttempts,
		"time_attempts":  s.TimeAttempts.String(),
		"time_await":     s.TimeAwait.String(),
		"resources":      s.Resources,
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
func (s *Flow) SetTimeAwait(time time.Duration, userId string) *Flow {
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
* getStep
* @param stepId string
* @return *Step, bool
**/
func (s *Flow) getStep(stepId string) (*Step, bool) {
	step, exists := s.Steps[stepId]
	if exists {
		return step, true
	}

	return nil, false
}

/**
* getOutput
* @param stepId string, index int
* @return *Connection, bool
**/
func (s *Flow) getOutput(stepId string, index int) (*Connection, bool) {
	for _, connection := range s.Connections {
		if connection.Kind == PortOutput && connection.Source.StepId == stepId && connection.Source.Index == index {
			return connection, true
		}
	}

	return nil, false
}

/**
* getError
* @param stepId string
* @return *Connection, bool
**/
func (s *Flow) getError(stepId string) (*Connection, bool) {
	for _, connection := range s.Connections {
		if connection.Kind == PortError && connection.Source.StepId == stepId && connection.Source.Index == 0 {
			return connection, true
		}
	}

	return nil, false
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
* @param tag, version, title string, kind Port, fn fnStep
* @return *Flow
**/
func (s *Flow) addStep(kind Kind, tag, version, title string, port Port, fn fnStep) *Flow {
	result := newStep(s.ID, "", kind, tag, version, title)
	result.fn = fn
	s.Steps[result.ID] = result

	if s.step == nil {
		s.step = result
		return s
	}

	_, exists := s.addConnection(s.step.ID, result.ID, 0, port)
	if !exists {
		return s
	}

	s.step = result

	return s
}

/**
* Step
* @param tag, version, title string, fn fnStep, userId string
* @return *Flow
**/
func (s *Flow) Step(tag, title string, fn fnStep) *Flow {
	if len(s.Steps) == 0 {
		result := newStep(s.ID, "", KindTrigger, tag, "1.0.0", title)
		result.fn = fn
		s.Steps[result.ID] = result
		s.step = result

		s.Triggers = append(s.Triggers, &Trigger{
			Tag:     tag,
			StartId: result.ID,
		})
		return s
	}

	return s.addStep(KindAction, tag, "1.0.0", title, PortOutput, fn)
}

/**
* Error
* @param stepId string
* @return *Flow
**/
func (s *Flow) Error(tag, version, title string, fn fnStep) *Flow {
	if len(s.Steps) == 0 {
		return s
	}

	return s.addStep(KindAction, tag, version, title, PortError, fn)
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
