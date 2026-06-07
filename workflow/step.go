package workflow

import (
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
)

type FnContext func(flow *Instance, ctx et.Json) (et.Json, error)

type Condition struct {
	Expression string `json:"expression"`
	YesTo      int    `json:"yes_to"`
	NoTo       int    `json:"no_to"`
}

type StParams struct {
	Name        string    `json:"-"`
	Description string    `json:"-"`
	Definition  string    `json:"-"`
	Undo        string    `json:"-"`
	Fn          FnContext `json:"-"`
	Stop        bool      `json:"-"`
}

type RefRollback struct {
	Definition string    `json:"-"`
	Fn         FnContext `json:"-"`
}

type Step struct {
	Index       int        `json:"index"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Stop        bool       `json:"stop"`
	Definition  []byte     `json:"definition"`
	Undo        []byte     `json:"undo"`
	fn          FnContext  `json:"-"`
	fnUndo      FnContext  `json:"-"`
	jrex        *jrex.Jrex `json:"-"`
	flow        *Flow      `json:"-"`
}

/**
* newStep
* @param flow *Flow, def StParams
* @return *Step
**/
func newStep(flow *Flow, def StParams) *Step {
	result := &Step{
		Name:        def.Name,
		Description: def.Description,
		Stop:        def.Stop,
		Definition:  []byte(def.Definition),
		Undo:        []byte{},
		fn:          def.Fn,
		flow:        flow,
	}
	flow.AddStep(result)
	return result
}

/**
* up
* @param flow *Flow
**/
func (s *Step) up(flow *Flow) {
	s.flow = flow
}

/**
* ToJson
* @return et.Json
**/
func (s *Step) ToJson() et.Json {
	return et.Json{
		"index":       s.Index,
		"name":        s.Name,
		"description": s.Description,
		"stop":        s.Stop,
		"definition":  string(s.Definition),
		"undo":        string(s.Undo),
	}
}

/**
* Set
* @param def StParams
* @return (*Step, error)
**/
func (s *Step) Set(def StParams) *Step {
	s.Name = def.Name
	s.Description = def.Description
	s.Stop = def.Stop
	s.Definition = []byte(def.Definition)
	s.fn = def.Fn
	return s
}

/**
* Rollback
* @params def RefRollback
* @return *Step
**/
func (s *Step) Rollback(def RefRollback) *Step {
	s.fnUndo = def.Fn
	s.Undo = []byte(def.Definition)
	return s
}

/**
* loadVm
* @params ctx et.Json
* @return *jrex.Jrex
**/
func (s *Step) loadVm(ctx et.Json) *jrex.Jrex {
	s.jrex = jrex.New("workflow", s.flow.workflow.store)
	s.jrex.SetCtx(ctx)
	return s.jrex
}

/**
* Run
* @params flow *Instance, ctx et.Json
* @return et.Json, error
**/
func (s *Step) Run(flow *Instance, ctx et.Json) (et.Json, error) {
	var result et.Json
	var err error
	defer func() {
		flow.setResult(result, err)
	}()

	flow.setStatus(RUNNING)
	if s.fn != nil {
		result, err = s.fn(flow, ctx)
	} else {
		s.jrex = s.loadVm(ctx)
		_, err = s.jrex.RunByBt(s.Definition)
		if err != nil {
			return nil, err
		}
		maps.Copy(result, ctx)
	}
	return result, err
}

/**
* RunRollback
* @params flow *Instance, ctx et.Json
* @return et.Json, error
**/
func (s *Step) RunRollback(flow *Instance, ctx et.Json) (et.Json, error) {
	var result et.Json
	var err error
	defer func() {
		flow.setRollback(result, err)
	}()

	flow.setStatus(ROLLBACK)
	if s.fnUndo != nil {
		result, err = s.fnUndo(flow, ctx)
	} else {
		s.jrex = s.loadVm(ctx)
		_, err = s.jrex.RunByBt(s.Undo)
		if err != nil {
			return nil, err
		}
		maps.Copy(result, ctx)
	}
	return result, err
}
