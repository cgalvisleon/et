package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type StepFn func(inst *Instance, ctx et.Json) (et.Json, error)
type Kind string

const (
	KindFunction  Kind = "function"
	KindTrigger   Kind = "trigger"
	KindAction    Kind = "action"
	KindCondition Kind = "condition"
)

var StepStatusList map[Status]bool = map[Status]bool{
	SYSTEM:   true,
	ACTIVE:   true,
	ARCHIVED: true,
	CANCEL:   true,
}

type Step struct {
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	TenantId    string         `json:"tenant_id"`
	ProjectId   string         `json:"project_id"`
	ID          string         `json:"id"`
	Version     string         `json:"version"`
	Type        string         `json:"type"`
	Kind        Kind           `json:"kind"`
	Tag         string         `json:"tag"`
	Status      Status         `json:"status"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Definition  interface{}    `json:"definition"`
	Config      et.Json        `json:"config"`
	Params      et.Json        `json:"params"`
	Inputs      int            `json:"inputs"`
	Outputs     int            `json:"outputs"`
	Stop        bool           `json:"stop"`
	AuditLog    []et.Json      `json:"audit_log"`
	UserID      string         `json:"-"`
	isDebug     bool           `json:"-"`
	isChanged   bool           `json:"-"`
	store       Store          `json:"-"`
	bindings    map[string]any `json:"-"`
}

type StepParams struct {
	TenantId    string      `json:"tenant_id"`
	ProjectId   string      `json:"project_id"`
	Version     string      `json:"version"`
	Type        string      `json:"type"`
	Kind        Kind        `json:"kind"`
	Tag         string      `json:"tag"`
	Status      Status      `json:"status"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Definition  interface{} `json:"definition"`
	Config      et.Json     `json:"config"`
	Params      et.Json     `json:"params"`
	Inputs      int         `json:"inputs"`
	Outputs     int         `json:"outputs"`
	Stop        bool        `json:"stop"`
	UserID      string      `json:"user_id"`
}

/**
* newStep
* @param def StepParams
* @return *Step
**/
func (s *WorkFlow) newStep(def StepParams) (*Step, error) {
	now := timezone.Now()
	if def.Version == "" {
		def.Version = "1.0.0"
	}
	def.Type = utility.Normalize(def.Type)
	def.Tag = utility.Normalize(def.Tag)
	id := fmt.Sprintf("step:%s:%s:%s", def.Kind, def.Type, def.Version)
	if def.Tag != "" {
		id = fmt.Sprintf("step:%s:%s:%s:%s", def.Kind, def.Type, def.Tag, def.Version)
	}

	if !StepStatusList[def.Status] {
		return nil, errors.New(MSG_STEP_STATUS_INVALID)
	}

	result := &Step{
		CreatedAt:   now,
		UpdatedAt:   now,
		TenantId:    def.TenantId,
		ProjectId:   def.ProjectId,
		ID:          id,
		Type:        def.Type,
		Kind:        def.Kind,
		Tag:         def.Tag,
		Version:     def.Version,
		Title:       def.Title,
		Description: def.Description,
		Definition:  def.Definition,
		Config:      def.Config,
		Params:      def.Params,
		Inputs:      def.Inputs,
		Outputs:     def.Outputs,
		Stop:        def.Stop,
		isDebug:     s.isDebug,
		AuditLog:    make([]et.Json, 0),
		UserID:      def.UserID,
		store:       s.store,
		bindings:    s.bindings,
	}
	result.AuditLog = append(result.AuditLog, et.Json{
		"created_at": now,
		"user_id":    def.UserID,
		"action":     "create",
	})
	return result, nil
}

/**
* loadStep
* @param id, userId string
* @return *Step, error
**/
func (s *WorkFlow) loadStep(id, userId string) (*Step, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	result := &Step{}
	exists, err := s.store.GetByCollection("step", id, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_STEP_NOT_FOUND)
	}

	result.store = s.store
	result.isDebug = s.isDebug
	result.bindings = s.bindings
	result.UserID = userId
	return result, nil
}

/**
* save
* @return error
**/
func (s *Step) save() error {
	if s.Kind == KindFunction {
		return errors.New(MSG_STEP_IS_FUNCTION)
	}

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

	return s.store.Set("step", s.ID, s.TenantId, s.ProjectId, s, s.UserID)
}

/**
* delete
* @return error
**/
func (s *Step) delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	return s.store.DeleteByCollection("step", s.ID)
}

/**
* ToJson
* @return et.Json
**/
func (s *Step) ToJson() et.Json {
	return et.Json{
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":   s.TenantId,
		"id":          s.ID,
		"type":        s.Type,
		"kind":        s.Kind,
		"version":     s.Version,
		"title":       s.Title,
		"description": s.Description,
		"definition":  s.Definition,
		"config":      s.Config,
		"params":      s.Params,
		"inputs":      s.Inputs,
		"outputs":     s.Outputs,
		"stop":        s.Stop,
	}
}

/**
* ToString
* @return string
**/
func (s *Step) ToString() string {
	return s.ToJson().ToString()
}

/**
* run
* @param instance *Instance, ctx et.Json
* @return error
**/
func (s *Step) run(instance *Instance, ctx et.Json) (et.Json, error) {
	switch v := s.Definition.(type) {
	case StepFn:
		instance.setStatus(RUNNING)
		result, err := v(instance, ctx)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	case string:
		jrex, err := jrex.New(s.ID, s.store)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		for name, binding := range instance.bindings {
			jrex.Set(name, binding)
		}
		instance.setStatus(RUNNING)
		jrex.SetCtx(ctx)
		result, err := jrex.RunByCode(v)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	case []byte:
		jrex, err := jrex.New(s.ID, s.store)
		if err != nil {
			return et.Json{}, err
		}
		for name, binding := range instance.bindings {
			jrex.Set(name, binding)
		}
		instance.setStatus(RUNNING)
		jrex.SetCtx(ctx)
		result, err := jrex.RunByBt(v)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	case []string:
		if instance.CurrentIndex < 0 || instance.CurrentIndex >= len(v) {
			return et.Json{}, errors.New(MSG_STEP_CODE_INDEX_NOT_FOUND)
		}
		code := v[instance.CurrentIndex]
		jrex, err := jrex.New(s.ID, s.store)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		for name, binding := range instance.bindings {
			jrex.Set(name, binding)
		}
		instance.setStatus(RUNNING)
		jrex.SetCtx(ctx)
		result, err := jrex.RunByCode(code)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	case [][]byte:
		if instance.CurrentIndex < 0 || instance.CurrentIndex >= len(v) {
			return et.Json{}, errors.New(MSG_STEP_CODE_INDEX_NOT_FOUND)
		}
		code := v[instance.CurrentIndex]
		jrex, err := jrex.New(s.ID, s.store)
		if err != nil {
			return et.Json{}, err
		}
		for name, binding := range instance.bindings {
			jrex.Set(name, binding)
		}
		instance.setStatus(RUNNING)
		jrex.SetCtx(ctx)
		result, err := jrex.RunByBt(code)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	}

	return et.Json{}, errors.New(MSG_STEP_DEFINITION_IS_UNKNOWN)
}

/**
* put
* @param version, title, description string, definition interface{}, config et.Json, params et.Json
* @return error
**/
func (s *Step) put(version, title, description string, definition interface{}, config et.Json, params et.Json) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	actions := s.AuditLog
	if s.Version != version {
		actions = append(actions, et.Json{
			"action": "update version",
			"old":    s.Version,
			"new":    version,
		})
		s.Version = version
		s.ID = fmt.Sprintf("step:%s:%s:%s", s.Kind, s.Type, s.Version)
		s.isChanged = true
	}

	if s.Title != title {
		actions = append(actions, et.Json{
			"action": "update title",
			"old":    s.Title,
			"new":    title,
		})
		s.Title = title
		s.isChanged = true
	}

	if s.Description != description {
		actions = append(actions, et.Json{
			"action": "update description",
			"old":    s.Description,
			"new":    description,
		})
		s.Description = description
		s.isChanged = true
	}

	if s.Definition != definition {
		actions = append(actions, et.Json{
			"action": "update definition",
		})
		s.Definition = definition
		s.isChanged = true
	}

	if s.isChanged {
		now := timezone.Now()
		s.AuditLog = append(s.AuditLog, et.Json{
			"created_at": now,
			"user_id":    s.UserID,
			"action":     actions,
		})
	}

	return s.save()
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Step) setStatus(status Status) error {
	if !StepStatusList[status] {
		return errors.New(MSG_STEP_STATUS_INVALID)
	}

	actions := s.AuditLog
	if s.Status == status {
		actions = append(actions, et.Json{
			"action": "update status",
			"old":    s.Status,
			"new":    status,
		})
		s.Status = status
		s.isChanged = true
	}

	return s.save()
}
