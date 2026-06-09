package workflow

import "github.com/cgalvisleon/et/et"

type Steper struct {
	Index       int    `json:"index"`
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       []int  `json:"steps"`
	flow        *Flow  `json:"-"`
}

/**
* newSteper
* @param flow *Flow, tag, name, description string
* @return *Steper
**/
func newSteper(flow *Flow, tag, name, description string) *Steper {
	result := &Steper{
		Tag:         tag,
		Name:        name,
		Description: description,
		Steps:       make([]int, 0),
		flow:        flow,
	}
	flow.Steper[tag] = result
	result.Index = len(flow.Steper) - 1
	return result
}

/**
* up
* @param flow *Flow
* @return void
**/
func (s *Steper) up(flow *Flow) {
	s.flow = flow
}

/**
* ToJson
* @return et.Json
**/
func (s *Steper) ToJson() et.Json {
	return et.Json{
		"index":       s.Index,
		"tag":         s.Tag,
		"name":        s.Name,
		"description": s.Description,
		"steps":       s.Steps,
	}
}

/**
* Step
* @param def StParams
* @return *Step
**/
func (s *Steper) Step(def StParams) *Step {
	result := newStep(s.flow, def)
	s.Steps = append(s.Steps, result.Index)
	return result
}

/**
* Rollback
* @param def RefRollback
* @return *Steper
**/
func (s *Steper) Rollback(def RefRollback) *Steper {
	idx := len(s.Steps)
	index := s.Steps[idx-1]
	step := s.flow.Steps[index]
	if step == nil {
		return nil
	}
	step.Rollback(def)
	return s
}

/**
* GetStep
* @param idx int
* @return (*Step, bool)
**/
func (s *Steper) GetStep(idx int) (*Step, bool) {
	if idx < 0 || idx >= len(s.Steps) {
		return nil, false
	}

	index := s.Steps[idx]
	result := s.flow.Steps[index]
	return result, result != nil
}
