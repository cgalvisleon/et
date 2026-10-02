package jwf

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Kind string

const (
	KindFunction  Kind = "function"
	KindTrigger   Kind = "trigger"
	KindAction    Kind = "action"
	KindCondition Kind = "condition"
	KindDelay     Kind = "delay"
	KindBucle     Kind = "bucle"
	KindIAAgent   Kind = "ia_agent"
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

type fnPublish func(flow *Flow, ctx et.Json) (et.Json, error)

type Step struct {
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	OwnerId     string                     `json:"owner_id"`
	ID          string                     `json:"id"`
	Kind        Kind                       `json:"kind"`
	Tag         string                     `json:"tag"`
	Version     string                     `json:"version"`
	Status      Status                     `json:"status"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Definition  string                     `json:"definition"`
	OnPublish   string                     `json:"on_publish"`
	Config      et.Json                    `json:"config"`
	Params      et.Json                    `json:"params"`
	Inputs      int                        `json:"inputs"`
	Outputs     int                        `json:"outputs"`
	Stop        bool                       `json:"stop"`
	AuditLog    []et.Json                  `json:"audit_log"`
	fn          fnStep                     `json:"-"`
	onPublish   fnPublish                  `json:"-"`
	isDebug     bool                       `json:"-"`
	isChanged   bool                       `json:"-"`
	bindings    map[string]any             `json:"-"`
	onChange    []func(data et.Json) error `json:"-"`
}

/**
* newStep
* @param ownerId, id string, kind Kind, tag, version, name string
* @return *Step
**/
func newStep(ownerId, id string, kind Kind, tag, version, name string) *Step {
	if version == "" {
		version = "1.0.0"
	}

	id = reg.GetUUID(id)
	now := timezone.Now()
	result := &Step{
		CreatedAt:   now,
		UpdatedAt:   now,
		OwnerId:     ownerId,
		ID:          id,
		Kind:        kind,
		Tag:         tag,
		Version:     version,
		Status:      ACTIVE,
		Name:        name,
		Description: "",
		Definition:  "",
		OnPublish:   "",
		Config:      et.Json{},
		Params:      et.Json{},
		Inputs:      0,
		Outputs:     1,
		Stop:        false,
		AuditLog:    make([]et.Json, 0),
		bindings:    make(map[string]any),
		onChange:    make([]func(data et.Json) error, 0),
	}
	return result.Up()
}

/**
* up
* @return *Step
**/
func (s *Step) Up() *Step {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Step) ToJson() et.Json {
	return et.Json{
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"owner_id":    s.OwnerId,
		"id":          s.ID,
		"kind":        s.Kind,
		"tag":         s.Tag,
		"version":     s.Version,
		"status":      s.Status,
		"name":        s.Name,
		"description": s.Description,
		"definition":  s.Definition,
		"on_publish":  s.OnPublish,
		"config":      s.Config,
		"params":      s.Params,
		"inputs":      s.Inputs,
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
* addAuditLog
* @param userId string, action string
**/
func (s *Step) addAuditLog(userId string, action string) {
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
* @return *Step
**/
func (s *Step) OnChange(fn func(data et.Json) error) *Step {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	s.onChange = append(s.onChange, fn)
	return s
}

/**
* run
* @param instance *Instance, ctx et.Json
* @return error
**/
func (s *Step) run(instance *Instance, ctx et.Json) (et.Json, error) {
	if s.fn != nil {
		instance.setStatus(RUNNING)
		result, err := s.fn(instance, ctx)
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return result, nil
	}

	initJrex := func() *jrex.Instance {
		result := jrex.NewInstance()
		for name, binding := range instance.bindings {
			result.Set(name, binding)
		}
		result.Set("params", s.Params)
		return result
	}

	runJrex := func(rex *jrex.Instance, script string) (et.Json, error) {
		instance.setStatus(RUNNING)
		if script == "" {
			return et.Json{}, nil
		}
		rex.SetCtx(ctx)
		rex.SetCode(script)
		_, err := rex.Run()
		if err != nil {
			instance.setStatus(FAILED)
			return et.Json{}, err
		}
		return rex.Ctx, nil
	}

	rex := initJrex()
	return runJrex(rex, s.Definition)
}

/**
* run
* @param instance *Instance, ctx et.Json
* @return error
**/
func (s *Step) runOnPublish(flow *Flow, ctx et.Json) (et.Json, error) {
	if s.onPublish != nil {
		result, err := s.onPublish(flow, ctx)
		if err != nil {
			return et.Json{}, err
		}
		return result, nil
	}

	initJrex := func() *jrex.Instance {
		result := jrex.NewInstance()
		result.Set("params", s.Params)
		result.Set("flow", flow)
		return result
	}

	runJrex := func(rex *jrex.Instance, script string) (et.Json, error) {
		rex.SetCtx(ctx)
		rex.SetCode(script)
		_, err := rex.Run()
		if err != nil {
			return et.Json{}, err
		}
		return rex.Ctx, nil
	}

	rex := initJrex()
	return runJrex(rex, s.OnPublish)
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

	if s.Status == status {
		return nil
	}

	s.Status = status
	s.addAuditLog(userId, fmt.Sprintf("update status: %s", status))
	return nil
}

/**
* setDefinition
* @param definition string
* @return *Step
**/
func (s *Step) setDefinition(definition string) *Step {
	s.Definition = definition
	return s
}

/**
* setOnPublish
* @param onPublish interface{}
* @return *Step
**/
func (s *Step) setOnPublish(onPublish string) *Step {
	s.OnPublish = onPublish
	return s
}
