package workflow

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/resilience"
)

type WorkFlow struct {
	Flows       map[string]*Flow       `json:"flows"`
	Instances   map[string]*Instance   `json:"instances"`
	Results     map[string]et.Json     `json:"results"`
	muFlows     sync.Mutex             `json:"-"`
	muInstances sync.Mutex             `json:"-"`
	store       instances.Store        `json:"-"`
	resilience  *resilience.Resilience `json:"-"`
	isDebug     bool                   `json:"-"`
}

var (
	packageName           = "workflow"
	ErrorInstanceNotFound = fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	workflow              *WorkFlow
)

/**
* Load
* @return error
**/
func Load(store instances.Store) error {
	if workflow != nil {
		return nil
	}

	err := event.Load()
	if err != nil {
		return err
	}

	resetInstance, err := resilience.New(store)
	if err != nil {
		return err
	}

	workflow = &WorkFlow{
		Flows:       make(map[string]*Flow),
		Instances:   make(map[string]*Instance),
		Results:     make(map[string]et.Json),
		muFlows:     sync.Mutex{},
		muInstances: sync.Mutex{},
		store:       store,
		resilience:  resetInstance,
		isDebug:     envar.GetBool("DEBUG", false),
	}

	return nil
}

/**
* addFlow
* @param flow *Flow
**/
func (s *WorkFlow) addFlow(flow *Flow) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	s.Flows[flow.Tag] = flow
}

/**
* getFlow
* @param tag string
* @return *Flow, bool
**/
func (s *WorkFlow) getFlow(tag string) (*Flow, bool) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	result, exists := s.Flows[tag]
	if exists {
		return result, true
	}

	if s.store != nil {
		exists, err := s.store.Get(tag, result)
		if err != nil {
			return nil, false
		}

		if exists {
			result.up(s)
			s.addFlow(result)
			return result, true
		}
	}

	return nil, false
}

/**
* removeFlow
* @param tag string
* @return bool
**/
func (s *WorkFlow) removeFlow(tag string) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	delete(s.Flows, tag)
}

/**
* CountFlows
* @return int
**/
func (s *WorkFlow) CountFlows() int {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	return len(s.Flows)
}

/**
* addInstance
* @param instance *Instance
**/
func (s *WorkFlow) addInstance(instance *Instance) {
	s.muInstances.Lock()
	defer s.muInstances.Unlock()

	s.Instances[instance.ID] = instance
}

/**
* getInstance
* @param id string
* @return *Instance, bool
**/
func (s *WorkFlow) getInstance(id string) (*Instance, bool) {
	s.muInstances.Lock()
	result, exists := s.Instances[id]
	s.muInstances.Unlock()

	if exists {
		return result, true
	}

	if s.store != nil {
		exists, err := s.store.Get(id, result)
		if err != nil {
			return nil, false
		}

		flow, ok := s.getFlow(result.Tag)
		if !ok {
			return nil, false
		}

		if exists {
			result.up(flow)
			result.isDebug = s.isDebug
			s.addInstance(result)
			return result, true
		}
	}

	return nil, false
}

/**
* removeInstance
* @param id string
**/
func (s *WorkFlow) removeInstance(id string) {
	s.muInstances.Lock()
	defer s.muInstances.Unlock()

	delete(s.Instances, id)
}

/**
* CountInstances
* @return int
**/
func (s *WorkFlow) CountInstances() int {
	s.muInstances.Lock()
	defer s.muInstances.Unlock()

	return len(s.Instances)
}

/**
* newInstance
* @param tag, id string, tags et.Json, step int, username string
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(tag, id string, tags et.Json, step int, username string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	flow := s.Flows[tag]
	if flow == nil {
		return nil, fmt.Errorf(MSG_FLOW_NOT_FOUND)
	}

	steper, ok := flow.Steper[tag]
	if !ok {
		return nil, fmt.Errorf(MSG_INVALID_STEPER_TAG, tag)
	}

	result := newInstance(steper, id, username)
	result.SetCurrentStep(step)
	result.putTag(tags)
	s.addInstance(result)

	return result, nil
}

/**
* GetInstance
* @param id string
* @return *Instance, error
**/
func (s *WorkFlow) GetInstance(id string) (*Instance, error) {
	result, exist := s.getInstance(id)
	if exist {
		return result, nil
	}

	return nil, ErrorInstanceNotFound
}

/**
* RunInstance
* @param id, tag string, step int, ctx, tags et.Json, username string
* @return et.Json, error
**/
func (s *WorkFlow) RunInstance(id, tag string, step int, ctx, tags et.Json, username string) (et.Json, error) {
	var err error
	instance, exist := s.getInstance(id)
	if !exist {
		instance, err = s.newInstance(tag, id, tags, step, username)
		if err != nil {
			return et.Json{}, err
		}
	} else {
		instance.putTag(tags)
	}

	instance.UpdatedBy = username
	instance.SetCurrentStep(step)
	instance.Stop(false)
	var result et.Json
	result, err = instance.run(ctx)
	if err != nil {
		return et.Json{}, err
	}

	s.removeInstance(instance.ID)
	if s.isDebug {
		logs.Debugf("instance: %s", instance.ToString())
	}

	return result, err
}

/**
* ResetInstance
* @param id string, username string
* @return et.Json, error
**/
func (s *WorkFlow) ResetInstance(id, username string) (et.Json, error) {
	instance, exists := s.getInstance(id)
	if !exists {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	}

	instance.UpdatedBy = username
	instance.SetCurrentStep(0)
	instance.setStatus(PENDING)
	return instance.ToJson()
}

/**
* RollbackInstance
* @param id string, username string
* @return et.Json, error
**/
func (s *WorkFlow) RollbackInstance(id, username string) (et.Json, error) {
	instance, exists := s.getInstance(id)
	if !exists {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	}

	instance.UpdatedBy = username
	result, err := instance.rollback(instance.Ctx)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* StopInstance
* @param instanceId string, username string
* @return et.Json, error
**/
func (s *WorkFlow) StopInstance(instanceId, username string) (et.Json, error) {
	instance, exists := s.getInstance(instanceId)
	if !exists {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	}

	instance.UpdatedBy = username
	err := instance.Stop(true)
	if err != nil {
		return et.Json{}, err
	}
	return instance.ToJson()
}

/**
* StatusInstance
* @param id, status, username string
* @return et.Json, error
**/
func (s *WorkFlow) StatusInstance(id, status, username string) (et.Json, error) {
	if _, ok := FlowStatusList[Status(status)]; !ok {
		return et.Json{}, fmt.Errorf(MSG_STATUS_INVALID, status)
	}

	instance, exists := s.getInstance(id)
	if !exists {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	}

	instance.UpdatedBy = username
	instance.SetStatus(Status(status))
	return instance.ToJson()
}

/**
* NewFlow
* @param tag, version, name, description, username string
* @return *Flow, error
**/
func (s *WorkFlow) NewFlow(tag, version, name, description, username string) (*Flow, error) {
	_, exists := s.getFlow(tag)
	if exists {
		return nil, errors.New(MSG_FLOW_ALREADY_EXISTS)
	}

	result := newFlow(tag, version, name, description, username)
	s.Flows[tag] = result
	return result, nil
}

/**
* DeleteFlow
* @param tag string
* @return error
**/
func (s *WorkFlow) DeleteFlow(tag string) error {
	flow, exists := s.getFlow(tag)
	if !exists {
		return errors.New(MSG_FLOW_NOT_FOUND)
	}

	err := flow.delete()
	if err != nil {
		return err
	}

	event.Publish(EVENT_WORKFLOW_DELETE, et.Json{
		"tag": flow.Tag,
	})
	delete(s.Flows, tag)
	return nil
}

/**
* NewSteper
* @param tag, name, description strin
* @return et.Json, error
**/
func (s *WorkFlow) NewSteper(flowTag, tag, name, description string) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	steper, err := flow.NewSteper(tag, name, description)
	if err != nil {
		return et.Json{}, err
	}

	return steper.ToJson(), nil
}

/**
* SetSteper
* @param flowTag, tag, name, description string
* @return et.Json, error
**/
func (s *WorkFlow) SetSteper(flowTag, tag, name, description string) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	steper, exist := flow.Steper[tag]
	if !exist {
		return et.Json{}, errors.New(MSG_STEPPER_NOT_FOUND)
	}

	steper.Name = name
	steper.Description = description
	err := flow.save()
	if err != nil {
		return et.Json{}, err
	}
	return steper.ToJson(), nil
}

/**
* DeleteSteper
* @param flowTag, tag string
* @return error
**/
func (s *WorkFlow) DeleteSteper(flowTag, tag string) error {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return errors.New(MSG_FLOW_NOT_FOUND)
	}

	_, exist := flow.Steper[tag]
	if !exist {
		return errors.New(MSG_STEPPER_NOT_FOUND)
	}

	delete(flow.Steper, tag)
	return flow.save()
}

/**
* AddStepFromSteper
* @param flowTag, tag string, index int
* @return *Flow, bool
**/
func (s *WorkFlow) AddStepFromSteper(flowTag, tag string, index int) (*Flow, bool) {
	flow, exists := s.Flows[flowTag]
	if !exists {
		return nil, false
	}

	steper, exist := flow.Steper[tag]
	if !exist {
		return nil, false
	}

	steper.Steps = append(steper.Steps, index)
	return flow, true
}

/**
* RemoveStepFromSteper
* @param flowTag, tag string, index int
* @return *Flow, bool
**/
func (s *WorkFlow) RemoveStepFromSteper(flowTag, tag string, index int) (*Flow, bool) {
	flow, exists := s.Flows[flowTag]
	if !exists {
		return nil, false
	}

	steper, exist := flow.Steper[tag]
	if !exist {
		return nil, false
	}

	steper.Steps = append(steper.Steps[:index], steper.Steps[index+1:]...)
	return flow, true
}

/**
* MoveStepFromSteper
* @param flowTag, tag string, index, to int
* @return et.Json, error
**/
func (s *WorkFlow) MoveStepFromSteper(flowTag, tag string, index, to int) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	step, exist := flow.Steper[tag]
	if !exist {
		return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
	}

	if to < 0 || to >= len(step.Steps) {
		return et.Json{}, errors.New(MSG_INVALID_TO_POSITION)
	}

	step.Steps = append(step.Steps[:index], step.Steps[index+1:]...)
	step.Steps = append(step.Steps[:to], append([]int{index}, step.Steps[to:]...)...)

	return step.ToJson(), nil
}

/**
* NewStep
* @param flowTag, steperTag, name, description, definition, undo string, stop bool
* @return et.Json, error
**/
func (s *WorkFlow) NewStep(flowTag, name, description, definition, undo string, stop bool) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	result, err := flow.NewStep(Def{
		Name:        name,
		Description: description,
		Definition:  definition,
		Undo:        undo,
		Stop:        stop,
	})
	if err != nil {
		return et.Json{}, err
	}

	return result.ToJson(), nil
}

/**
* SetStep
* @param flowTag, steperTag string, index int, name, description, definition, undo string, stop bool
* @return et.Json, error
**/
func (s *WorkFlow) SetStep(flowTag string, index int, name, description, definition, undo string, stop bool) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	step, err := flow.SetStep(index, name, description, definition, undo, stop)
	if err != nil {
		return et.Json{}, err
	}

	return step.ToJson(), nil
}

/**
* DeleteStep
* @param flowTag string, index int
* @return et.Json, error
**/
func (s *WorkFlow) DeleteStep(flowTag string, index int) (et.Json, error) {
	flow, exists := s.getFlow(flowTag)
	if !exists {
		return et.Json{}, errors.New(MSG_FLOW_NOT_FOUND)
	}

	flow.Steps = append(flow.Steps[:index], flow.Steps[index+1:]...)
	return et.Json{}, nil
}
