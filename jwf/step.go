package jwf

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
)

type Kind string

const (
	KindFunction        Kind = "function"
	KindTrigger         Kind = "trigger"
	KindAction          Kind = "action"
	KindCondition       Kind = "condition"
	KindDelay           Kind = "delay"
	KindBucle           Kind = "bucle"
	KindIAAgent         Kind = "ia_agent"
	LANGUAGE_JAVASCRIPT      = "javascript"
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
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	OwnerId     string              `json:"owner_id"`
	ID          string              `json:"id"`
	Kind        Kind                `json:"kind"`
	Tag         string              `json:"tag"`
	Version     string              `json:"version"`
	Status      Status              `json:"status"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Definition  []*Script           `json:"definition"`
	Config      et.Json             `json:"config"`
	Params      et.Json             `json:"params"`
	Inputs      int                 `json:"inputs"`
	Outputs     int                 `json:"outputs"`
	Stop        bool                `json:"stop"`
	fn          fnStep              `json:"-"`
	isDebug     bool                `json:"-"`
	isChanged   bool                `json:"-"`
	bindings    map[string]any      `json:"-"`
	onStatus    func(status Status) `json:"-"`
	onChange    func(data et.Json)  `json:"-"`
}

/**
* newStep
* @param ownerId, id string, kind Kind, tag, version, title string
* @return *Step
**/
func newStep(ownerId, id string, kind Kind, tag, version, title string) *Step {
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
		Title:       title,
		Description: "",
		Definition:  []*Script{},
		Config:      et.Json{},
		Params:      et.Json{},
		Inputs:      0,
		Outputs:     1,
		Stop:        false,
		bindings:    make(map[string]any),
	}
	return result
}

func LoadStep(def et.Json) (*Step, error) {
	bt, err := def.ToByte()
	if err != nil {
		return nil, err
	}

	var result *Step
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	return result.up(), nil
}

/**
* up
* @return *Step
**/
func (s *Step) up() *Step {
	if s.onStatus == nil {
		s.onStatus = func(status Status) {}
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
* OnStatus
* @param fn func(status Status)
**/
func (s *Step) OnStatus(fn func(status Status)) {
	s.onStatus = fn
}

/**
* OnChange
* @param fn func(data et.Json) error
**/
func (s *Step) OnChange(fn func(data et.Json)) {
	s.onChange = fn
}

/**
* setStatus
* @param status Status
**/
func (s *Step) setStatus(status Status) {
	if s.onStatus != nil {
		s.onStatus(status)
	}
	if s.onChange != nil {
		s.onChange(s.ToJson())
	}
}

/**
* SetBinding
* @param name string, value interface{}
* @return *Step
**/
func (s *Step) SetBinding(name string, value interface{}) *Step {
	if s.bindings == nil {
		s.bindings = make(map[string]any)
	}
	s.bindings[name] = value
	return s
}

/**
* RunFunction
* @param instance *Instance, ctx et.Json
* @return et.Json, error
**/
func (s *Step) RunFunction(instance *Instance, ctx et.Json) (et.Json, error) {
	if s.fn == nil {
		return et.Json{}, nil
	}
	s.setStatus(RUNNING)
	result, err := s.fn(instance, ctx)
	if err != nil {
		s.setStatus(FAILED)
		return et.Json{}, err
	}
	return result, nil
}

/**
* RunScript
* @param ctx et.Json
* @return et.Json, error
**/
func (s *Step) RunScript(ctx et.Json, instance *Instance) (et.Json, error) {
	result := et.Json{}
	s.setStatus(RUNNING)
	script := ""
	for _, scr := range s.Definition {
		script = strs.Append(script, scr.Code, "\n")
	}
	_, err := RunScript(script, ctx, s.bindings)
	if err != nil {
		s.setStatus(FAILED)
		return et.Json{}, err
	}
	return result, nil
}

/**
* Run
* @param instance *Instance, ctx et.Json
* @return error
**/
func (s *Step) Run(instance *Instance, ctx et.Json) (et.Json, error) {
	if s.fn != nil {
		return s.RunFunction(instance, ctx)
	}
	return s.RunScript(ctx, instance)
}
