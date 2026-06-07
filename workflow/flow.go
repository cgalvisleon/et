package workflow

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

type CheckList struct {
	Tag         string  `json:"tag"`
	Description string  `json:"description"`
	Ok          bool    `json:"ok"`
	Data        et.Json `json:"data"`
}

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

type Flow struct {
	TenantId      string             `json:"tenant_id"`
	OwnerId       string             `json:"owner_id"`
	Tag           string             `json:"tag"`
	Version       string             `json:"version"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Steps         []*Step            `json:"steps"`
	Steper        map[string]*Steper `json:"steeper"`
	CheckList     []*CheckList       `json:"check_list"`
	TotalAttempts int                `json:"total_attempts"`
	TimeAttempts  time.Duration      `json:"time_attempts"`
	Team          string             `json:"team"`
	Level         string             `json:"level"`
	CreatedBy     string             `json:"created_by"`
	UpdatedBy     string             `json:"updated_by"`
	workflow      *WorkFlow          `json:"-"`
	isDebug       bool               `json:"-"`
}

/**
* newFlow
* @param tenantId, ownerId, tag, version, name, description string, username string
* @return *Flow
**/
func newFlow(tenantId, ownerId, tag, version, name, description string, username string) *Flow {
	flow := &Flow{
		TenantId:    tenantId,
		OwnerId:     ownerId,
		Tag:         tag,
		Version:     version,
		Name:        name,
		Description: description,
		Steps:       make([]*Step, 0),
		Steper:      make(map[string]*Steper),
		CheckList:   make([]*CheckList, 0),
		CreatedBy:   username,
		isDebug:     envar.GetBool("DEBUG", false),
	}
	logs.Logf(packageName, MSG_FLOW_CREATED, tag, version, name)

	return flow
}

/**
* ToJson
* @return et.Json
**/
func (s *Flow) ToJson() et.Json {
	return et.Json{
		"tenant_id":      s.TenantId,
		"owner_id":       s.OwnerId,
		"tag":            s.Tag,
		"version":        s.Version,
		"name":           s.Name,
		"description":    s.Description,
		"steps":          s.Steps,
		"steper":         s.Steper,
		"checklist":      s.CheckList,
		"total_attempts": s.TotalAttempts,
		"time_attempts":  s.TimeAttempts.String(),
		"team":           s.Team,
		"level":          s.Level,
		"created_by":     s.CreatedBy,
	}
}

/**
* save
* @return error
**/
func (s *Flow) save(userId string) error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.workflow != nil && s.workflow.store != nil {
		err := s.workflow.store.Set(s.Tag, "flow", s.TenantId, s.OwnerId, s, userId)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_FLOW_SET, data)

	return nil
}

/**
* delete
* @return error
**/
func (s *Flow) delete() error {
	if s.workflow != nil && s.workflow.store != nil {
		err := s.workflow.store.Delete(s.Tag)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_FLOW_DELETE, et.Json{
		"tag": s.Tag,
	})

	return nil
}

/**
* up
* @param workflow *WorkFlow
**/
func (s *Flow) up(workflow *WorkFlow) {
	s.workflow = workflow
	s.isDebug = workflow.isDebug
	for _, step := range s.Steps {
		step.up(s)
	}
	for _, steper := range s.Steper {
		steper.up(s)
	}
}

/**
* Debug
**/
func (s *Flow) Debug() {
	s.isDebug = true
}

/**
* AddStep
* @param step *Step
**/
func (s *Flow) AddStep(step *Step) {
	s.Steps = append(s.Steps, step)
	step.Index = len(s.Steps) - 1
	logs.Logf(packageName, MSG_INSTANCE_STEP_CREATED, step.Index, step.Name, s.Tag)
}

/**
* NewSteper
* @param tag, name, description, userId string
* @return *Steper, error
**/
func (s *Flow) NewSteper(tag, name, description, userId string) (*Steper, error) {
	_, ok := s.Steper[tag]
	if ok {
		return nil, fmt.Errorf(MSG_STEPER_ALREADY_EXISTS, tag)
	}

	result := newSteper(s, tag, name, description)
	return result, s.save(userId)
}

/**
* GetSteper
* @param tag string
* @return (*Steper, error)
**/
func (s *Flow) GetSteper(tag string) (*Steper, error) {
	steper, ok := s.Steper[tag]
	if !ok {
		return nil, fmt.Errorf(MSG_INVALID_STEPER_TAG, tag)
	}

	return steper, nil
}

/**
* SetSteper
* @param tag, name, description, userId string
* @return (*Steper, error)
**/
func (s *Flow) SetSteper(tag, name, description, userId string) (*Steper, error) {
	if tag == "" {
		return nil, fmt.Errorf(MSG_INVALID_STEPER_TAG, tag)
	}

	steper, ok := s.Steper[tag]
	if !ok {
		return nil, fmt.Errorf(MSG_INVALID_STEPER_TAG, tag)
	}

	steper.Name = name
	steper.Description = description
	return steper, s.save(userId)
}

/**
* NewStep
* @param def StParams
* @return *Step, error
**/
func (s *Flow) NewStep(def StParams, userId string) (*Step, error) {
	step := &Step{
		Index:       len(s.Steps),
		Name:        def.Name,
		Description: def.Description,
		Definition:  []byte(def.Definition),
		Undo:        []byte(def.Undo),
		Stop:        def.Stop,
	}
	s.Steps = append(s.Steps, step)
	return step, s.save(userId)
}

/**
* SetStep
* @param index int, name, description, definition, undo string, stop bool
* @return (*Step, error)
**/
func (s *Flow) SetStep(index int, name, description, definition, undo string, stop bool, userId string) (*Step, error) {
	step := s.Steps[index]
	if step == nil {
		return nil, errors.New(MSG_STEP_NOT_FOUND)
	}

	step.Name = name
	step.Description = description
	step.Definition = []byte(definition)
	step.Undo = []byte(undo)
	step.Stop = stop
	return step, s.save(userId)
}

/**
* Step
* @param def StParams
* @return *Steper
**/
func (s *Flow) Step(def StParams) *Steper {
	result := newSteper(s, s.Tag, s.Name, s.Description)
	result.Step(def)
	return result
}

/**
* Resilence
* @param totalAttempts int, timeAttempts time.Duration
* @return *Flow
**/
func (s *Flow) Resilence(totalAttempts int, timeAttempts time.Duration, team string, level string) *Flow {
	s.TotalAttempts = totalAttempts
	s.TimeAttempts = timeAttempts
	s.Team = team
	s.Level = level
	logs.Logf(packageName, MSG_INSTANCE_RESILIENCE, s.Tag, totalAttempts, timeAttempts)
	return s
}

/**
* DefineCheckList
* @param tag string, description string, ok bool, data et.Json
* @return *Flow
**/
func (s *Flow) DefineCheckList(tag string, description string) *Flow {
	s.CheckList = append(s.CheckList, &CheckList{
		Tag:         tag,
		Description: description,
		Ok:          false,
		Data:        et.Json{},
	})
	return s
}

/**
* RemoveCheckList
* @param tag string
* @return *Flow
**/
func (s *Flow) RemoveCheckList(tag string) *Flow {
	idx := slices.IndexFunc(s.CheckList, func(check *CheckList) bool { return check.Tag == tag })
	if idx != -1 {
		s.CheckList = append(s.CheckList[:idx], s.CheckList[idx+1:]...)
	}
	return s
}
