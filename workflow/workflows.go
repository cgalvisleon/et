package workflow

import (
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

var (
	packageName           = "workflow"
	ErrorInstanceNotFound = fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
)

type WorkFlows struct {
	Flows     map[string]*Flow     `json:"flows"`
	Instances map[string]*Instance `json:"instances"`
	Results   map[string]et.Json   `json:"results"`
	mu        sync.Mutex           `json:"-"`
	isDebug   bool                 `json:"-"`
}

/**
* newWorkFlows
* @return *WorkFlows
**/
func newWorkFlows() *WorkFlows {
	result := &WorkFlows{
		Flows:     make(map[string]*Flow),
		Instances: make(map[string]*Instance),
		Results:   make(map[string]et.Json),
		mu:        sync.Mutex{},
		isDebug:   envar.GetBool("DEBUG", false),
	}

	return result
}

/**
* healthCheck
* @return bool
**/
func (s *WorkFlows) healthCheck() bool {
	ok := resilience.HealthCheck()
	if !ok {
		return false
	}

	return true
}

/**
* Add
* @param instance *Instance
**/
func (s *WorkFlows) Add(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Instances[instance.Id] = instance
}

/**
* Remove
* @param instance *Instance
**/
func (s *WorkFlows) Remove(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Instances, instance.Id)
}

/**
* Count
* @return int
**/
func (s *WorkFlows) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.Instances)
}

/**
* newInstance
* @param tag, id string, tags et.Json, step int, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) newInstance(tag, id string, tags et.Json, step int, createdBy string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	flow := s.Flows[tag]
	if flow == nil {
		return nil, fmt.Errorf(MSG_FLOW_NOT_FOUND)
	}

	if step == -1 {
		step = 0
	}

	now := timezone.Now()
	result := &Instance{
		Flow:       flow,
		workFlows:  s,
		Tag:        tag,
		CreatedAt:  now,
		UpdatedAt:  now,
		Id:         id,
		CreatedBy:  createdBy,
		UpdatedBy:  createdBy,
		Current:    step,
		Ctx:        et.Json{},
		Ctxs:       make(map[int]et.Json),
		Results:    make(map[int]*Result),
		Rollbacks:  make(map[int]*Result),
		Tags:       tags,
		goTo:       -1,
		WorkerHost: workerHost,
		Params:     et.Json{},
		isNew:      true,
	}
	result.setStatus(FlowStatusPending)

	return result, nil
}

/**
* loadInstance
* @param id string
* @return *Flow, error
**/
func (s *WorkFlows) loadInstance(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, ok := s.Instances[id]
	if ok {
		return result, true
	}

	if getInstance != nil {
		exists, err := getInstance(id, &result)
		if err != nil {
			return nil, false
		}

		if !exists {
			return nil, false
		}

		flow := s.Flows[result.Tag]
		if flow == nil {
			return nil, false
		}

		result.Flow = flow
		result.goTo = -1
		s.Add(result)

		if s.isDebug {
			logs.Log("WorkFlows", "loadInstance:", result.ToJson().ToString())
		}

		return result, true
	}

	return nil, false
}

/**
* getOrCreateInstance
* @param id, tag string, step int, tags et.Json, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) getOrCreateInstance(id, tag string, step int, tags et.Json, createdBy string) (*Instance, error) {
	id = reg.GetUUID(id)
	result, exists := s.loadInstance(id)
	if !exists {
		return s.newInstance(tag, id, tags, step, createdBy)
	}

	return result, nil
}

/**
* runInstance
* Si el step es -1 se ejecuta el siguiente paso, si no se ejecuta el paso indicado
* @param instanceId, tag string, step int, tags, ctx et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlows) runInstance(instanceId, tag string, step int, tags, ctx et.Json, createdBy string) (et.Json, error) {
	instance, err := s.getOrCreateInstance(instanceId, tag, step, tags, createdBy)
	if err != nil {
		return et.Json{}, err
	}

	instance.isDebug = s.isDebug
	instance.UpdatedBy = createdBy
	instance.PutTag(tags)
	if step != instance.Current {
		instance.Current = step
	}
	result, err := instance.run(ctx)
	if err != nil {
		return et.Json{}, err
	}

	s.Remove(instance)
	logs.Logf(packageName, "runInstance: %s", tag)
	if s.isDebug {
		logs.Debugf("instance: %s", instance.ToString())
	}

	return result, err
}

/**
* resetInstance
* @param instanceId string
* @return error
**/
func (s *WorkFlows) resetInstance(instanceId, updatedBy string) error {
	instance, exists := s.loadInstance(instanceId)
	if !exists {
		return fmt.Errorf("instance not found")
	}

	instance.UpdatedBy = updatedBy
	instance.setStatus(FlowStatusPending)

	return nil
}

/**
* Rollback
* @param instanceId string, updatedBy string
* @return et.Json, error
**/
func (s *WorkFlows) rollback(instanceId, updatedBy string) (et.Json, error) {
	instance, exists := s.loadInstance(instanceId)
	if !exists {
		return et.Json{}, fmt.Errorf("instance not found")
	}

	instance.UpdatedBy = updatedBy
	result, err := instance.rollback(et.Json{}, nil)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* stop
* @param instanceId string, updatedBy string
* @return error
**/
func (s *WorkFlows) stop(instanceId, updatedBy string) error {
	instance, exists := s.loadInstance(instanceId)
	if !exists {
		return fmt.Errorf("instance not found")
	}

	instance.UpdatedBy = updatedBy
	return instance.Stop()
}

/**
* newFlow
* @param tag, version, name, description string, fn FnContext, stop bool, createdBy string
* @return *Flow
**/
func (s *WorkFlows) newFlow(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := newFlow(tag, version, name, description, fn, stop, createdBy)
	s.Flows[tag] = flow

	return flow
}

/**
* deleteFlow
* @param tag string
* @return bool
**/
func (s *WorkFlows) deleteFlow(tag string) bool {
	if s.Flows[tag] == nil {
		return false
	}

	flow := s.Flows[tag]
	event.Publish(EVENT_WORKFLOW_DELETE, flow.ToJson())
	delete(s.Flows, tag)

	return true
}
