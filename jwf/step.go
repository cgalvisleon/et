package jwf

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type StepFn func(inst *Instance, ctx et.Json) (et.Json, error)
type Kind string

const (
	KindFunction  Kind = "function"
	KindTrigger   Kind = "trigger"
	KindAction    Kind = "action"
	KindCondition Kind = "condition"
	KindDelay     Kind = "delay"
)

var (
	ErrrStepNotFound               = errors.New(MSG_STEP_NOT_FOUND)
	KindList         map[Kind]bool = map[Kind]bool{
		KindFunction:  true,
		KindTrigger:   true,
		KindAction:    true,
		KindCondition: true,
	}

	StepStatusList map[Status]bool = map[Status]bool{
		SYSTEM:   true,
		ACTIVE:   true,
		ARCHIVED: true,
		CANCEL:   true,
	}
)

type Step struct {
	CreatedAt   time.Time                               `json:"created_at"`
	UpdatedAt   time.Time                               `json:"updated_at"`
	TenantId    string                                  `json:"tenant_id"`
	ID          string                                  `json:"id"`
	Kind        Kind                                    `json:"kind"`
	Tag         string                                  `json:"tag"`
	Version     string                                  `json:"version"`
	Status      Status                                  `json:"status"`
	Title       string                                  `json:"title"`
	Description string                                  `json:"description"`
	Definition  interface{}                             `json:"definition"`
	Config      et.Json                                 `json:"config"`
	Params      et.Json                                 `json:"params"`
	Outputs     int                                     `json:"outputs"`
	Stop        bool                                    `json:"stop"`
	AuditLog    []et.Json                               `json:"audit_log"`
	isDebug     bool                                    `json:"-"`
	isChanged   bool                                    `json:"-"`
	store       Store                                   `json:"-"`
	bindings    map[string]any                          `json:"-"`
	onSave      []func(step *Step, userId string) error `json:"-"`
	onDelete    []func(step *Step, userId string) error `json:"-"`
}

/**
* newStep
* @param kind Kind, tag, version, title string
* @return *Step
**/
func (s *WorkFlow) newStep(kind Kind, tag, version, title string) (*Step, error) {
	if version == "" {
		version = "1.0.0"
	}

	now := timezone.Now()
	id := reg.ULID()
	result := &Step{
		CreatedAt: now,
		UpdatedAt: now,
		TenantId:  s.TenantId,
		ID:        id,
		Kind:      kind,
		Tag:       tag,
		Version:   version,
		Status:    ACTIVE,
		Title:     title,
		Config:    et.Json{},
		Params:    et.Json{},
		Stop:      false,
		AuditLog:  make([]et.Json, 0),
		isDebug:   s.isDebug,
		store:     s.store,
		bindings:  s.bindings,
		onSave:    make([]func(step *Step, userId string) error, 0),
		onDelete:  make([]func(step *Step, userId string) error, 0),
	}
	result.
		OnSave(func(step *Step, userId string) error {
			key := fmt.Sprintf("step:%s", step.ID)
			event.Publish(key, step.ToJson())
			return nil
		}).
		OnDelete(func(step *Step, userId string) error {
			key := fmt.Sprintf("step:%s:delete", step.ID)
			event.Publish(key, et.Json{
				"id": step.ID,
			})
			return nil
		})
	return result, nil
}

/**
* loadStep
* @param id, userId string
* @return *Step, error
**/
func (s *WorkFlow) getStep(id string) (*Step, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	var result *Step
	exists, err := s.store.Get("step", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrrStepNotFound
	}

	result.store = s.store
	result.isDebug = s.isDebug
	result.bindings = s.bindings
	result.onSave = make([]func(step *Step, userId string) error, 0)
	result.onDelete = make([]func(step *Step, userId string) error, 0)
	result.
		OnSave(func(step *Step, userId string) error {
			key := fmt.Sprintf("step:%s", step.ID)
			event.Publish(key, step.ToJson())
			return nil
		}).
		OnDelete(func(step *Step, userId string) error {
			key := fmt.Sprintf("step:%s:delete", step.ID)
			event.Publish(key, et.Json{
				"id": step.ID,
			})
			return nil
		})
	return result, nil
}

/**
* deleteStep
* @param id string
* @return error
**/
func (s *WorkFlow) deleteStep(id, userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	step, err := s.getStep(id)
	if err != nil {
		return err
	}

	err = s.store.Delete("step", id)
	if err != nil {
		return err
	}

	for _, onDelete := range step.onDelete {
		err := onDelete(step, userId)
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
func (s *Step) addAuditLog(userId string, action string) {
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
* OnSave
* @param fn func(step *Step, userId string) error
* @return *Jrex
**/
func (s *Step) OnSave(fn func(step *Step, userId string) error) *Step {
	if s.onSave == nil {
		s.onSave = make([]func(step *Step, userId string) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(step *Step, userId string) error
* @return *Step
**/
func (s *Step) OnDelete(fn func(step *Step, userId string) error) *Step {
	if s.onDelete == nil {
		s.onDelete = make([]func(step *Step, userId string) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Step) save(userId string) error {
	if s.Kind == KindFunction {
		return errors.New(MSG_STEP_IS_FUNCTION)
	}

	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("step", s.ID, s.TenantId, "", s, userId)
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
func (s *Step) ToJson() et.Json {
	return et.Json{
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":   s.TenantId,
		"id":          s.ID,
		"kind":        s.Kind,
		"tag":         s.Tag,
		"version":     s.Version,
		"title":       s.Title,
		"description": s.Description,
		"definition":  s.Definition,
		"config":      s.Config,
		"params":      s.Params,
		"outputs":     s.Outputs,
		"stop":        s.Stop,
		"audit_log":   s.AuditLog,
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
func (s *Step) run(instance *Instance, ctx et.Json, userId string) (et.Json, error) {
	if s.Definition == nil {
		return et.Json{}, errors.New(MSG_STEP_DEFINITION_IS_NIL)
	}

	initJrex := func() *jrex.Instance {
		result := jrex.NewInstance()
		for name, binding := range instance.bindings {
			result.Set(name, binding)
		}
		return result
	}

	runJrex := func(rex *jrex.Instance, script string) (et.Json, error) {
		instance.setStatus(RUNNING, userId)
		rex.SetCtx(ctx)
		_, err := rex.RunString(script)
		if err != nil {
			instance.setStatus(FAILED, userId)
			return et.Json{}, err
		}
		return rex.Ctx, nil
	}

	switch v := s.Definition.(type) {
	case StepFn:
		instance.setStatus(RUNNING, userId)
		result, err := v(instance, ctx)
		if err != nil {
			instance.setStatus(FAILED, userId)
			return et.Json{}, err
		}
		return result, nil
	case string:
		rex := initJrex()
		return runJrex(rex, v)
	case []byte:
		rex := initJrex()
		code := string(v)
		return runJrex(rex, code)
	case []string:
		rex := initJrex()
		if instance.CurrentIndex < 0 || instance.CurrentIndex >= len(v) {
			return et.Json{}, errors.New(MSG_STEP_CODE_INDEX_NOT_FOUND)
		}
		code := v[instance.CurrentIndex]
		return runJrex(rex, code)
	case [][]byte:
		rex := initJrex()
		if instance.CurrentIndex < 0 || instance.CurrentIndex >= len(v) {
			return et.Json{}, errors.New(MSG_STEP_CODE_INDEX_NOT_FOUND)
		}
		bt := v[instance.CurrentIndex]
		code := string(bt)
		return runJrex(rex, code)
	}

	return et.Json{}, errors.New(MSG_STEP_DEFINITION_IS_UNKNOWN)
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Step) setStatus(status Status, userId string) error {
	if !StepStatusList[status] {
		return errors.New(MSG_STEP_STATUS_INVALID)
	}

	if s.Status != status {
		s.addAuditLog(userId, fmt.Sprintf("update status: %s", status))
		s.Status = status
		return s.save(userId)
	}
	return nil
}

/**
* setDefinition
* @param definition interface{}
* @return error
**/
func (s *Step) setDefinition(definition interface{}, userId string) error {
	if s.Definition != definition {
		s.Definition = definition
		s.addAuditLog(userId, fmt.Sprintf("update definition"))
		return s.save(userId)
	}
	return nil
}

/**
* put
* @param version, title, description string, config et.Json, params et.Json, userId string
* @return error
**/
func (s *Step) put(version, title, description string, config et.Json, params et.Json, userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	if s.Version != version {
		s.addAuditLog(userId, fmt.Sprintf("update version old:%s", s.Version))
		s.Version = version
	}

	if s.Title != title {
		s.addAuditLog(userId, fmt.Sprintf("update title  ol:%s", s.Title))
		s.Title = title
	}

	if s.Description != description {
		s.addAuditLog(userId, fmt.Sprintf("update description old:%s", s.Description))
		s.Description = description
	}

	if s.Config.ToString() != config.ToString() {
		s.addAuditLog(userId, fmt.Sprintf("update config old:%s", s.Config))
		s.Config = config
	}

	if s.Params.ToString() != params.ToString() {
		s.addAuditLog(userId, fmt.Sprintf("update params old:%s", s.Params))
		s.Params = params
	}

	return s.save(userId)
}
