package workflow

import (
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/vm"
)

var (
	errorInstanceNotFound = fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
)

const packageName = "workflow"

type instanceFn func(instanceId, tag string, startId int, tags, ctx et.Json, createdBy string) (et.Json, error)

type WorkFlows struct {
	Flows     map[string]*Flow     `json:"flows"`
	Instances map[string]*Instance `json:"instances"`
	mu        sync.Mutex           `json:"-"`
}

/**
* newWorkFlows
* @return *WorkFlows
**/
func newWorkFlows() *WorkFlows {
	result := &WorkFlows{
		Flows:     make(map[string]*Flow),
		Instances: make(map[string]*Instance),
		mu:        sync.Mutex{},
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
**/
func (s *WorkFlows) Add(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Instances[instance.Id] = instance
}

/**
* Remove
**/
func (s *WorkFlows) Remove(instanceId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Instances, instanceId)
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
* @param tag, id string, tags et.Json, startId int, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) newInstance(tag, id string, tags et.Json, startId int, createdBy string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	flow := s.Flows[tag]
	if flow == nil {
		return nil, fmt.Errorf(MSG_FLOW_NOT_FOUND)
	}

	if startId == -1 {
		startId = 0
	}

	now := timezone.NowTime()
	result := &Instance{
		Flow:       flow,
		workFlows:  s,
		CreatedAt:  now,
		UpdatedAt:  now,
		Tag:        tag,
		Id:         id,
		CreatedBy:  createdBy,
		UpdatedBy:  createdBy,
		Current:    startId,
		Ctx:        et.Json{},
		Ctxs:       make(map[int]et.Json),
		PinnedData: et.Json{},
		Results:    make(map[int]*Result),
		Rollbacks:  make(map[int]*Result),
		Tags:       tags,
		WorkerHost: workerHost,
		goTo:       -1,
		vm:         vm.New(),
	}
	result.SetStatus(FlowStatusPending)
	s.Add(result)

	return result, nil
}

/**
* loadInstance
* @param id string
* @return *Flow, error
**/
func (s *WorkFlows) loadInstance(id string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	if loadInstance != nil {
		return loadInstance(id)
	}

	result, ok := s.Instances[id]
	if !ok {
		return nil, errorInstanceNotFound
	}

	return result, nil
}

/**
* getOrCreateInstance
* @param id, tag string, startId int, tags et.Json, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) getOrCreateInstance(id, tag string, startId int, tags et.Json, createdBy string) (*Instance, error) {
	id = reg.GetUUID(id)
	result, err := s.loadInstance(id)
	if err != nil {
		return s.newInstance(tag, id, tags, startId, createdBy)
	}

	return result, nil
}

/**
* run
* Si el step es -1 se ejecuta el siguiente paso, si no se ejecuta el paso indicado
* @param instanceId, tag string, step int, tags, ctx et.Json, runBy string
* @return et.Json, error
**/
func (s *WorkFlows) run(instanceId, tag string, step int, tags, ctx et.Json, runBy string) (et.Json, error) {
	instance, err := s.getOrCreateInstance(instanceId, tag, step, tags, runBy)
	if err != nil {
		return et.Json{}, err
	}

	instance.SetTags(tags)
	if step != -1 {
		instance.Current = step
		currentCtx := instance.Ctxs[step]
		instance.SetCtx(currentCtx)
	}
	result, err := instance.run(ctx, runBy)
	if err != nil {
		return et.Json{}, err
	}

	s.Remove(instanceId)
	if instance.isDebug {
		logs.Debugf("run InstanceId:%s:%s", instanceId, instance.ToJson().ToString())
	}

	return result, err
}

/**
* reset
* @param instanceId, updatedBy string
* @return error
**/
func (s *WorkFlows) reset(instanceId, updatedBy string) error {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return err
	}

	instance.UpdatedBy = updatedBy
	instance.SetStatus(FlowStatusPending)

	return nil
}

/**
* rollback
* @param instanceId string
* @return et.Json, error
**/
func (s *WorkFlows) rollback(instanceId string) (et.Json, error) {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return et.Json{}, err
	}

	result, err := instance.rollback(et.Json{}, nil)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* stop
* @param instanceId string
* @return error
**/
func (s *WorkFlows) stop(instanceId string) error {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return err
	}

	return instance.Stop()
}

/**
* delete
* @param instanceId string
* @return error
**/
func (s *WorkFlows) delete(instanceId string) error {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return err
	}

	if delInstance != nil {
		delInstance(instanceId)
	}

	s.Remove(instanceId)
	event.Publish(EVENT_WORKFLOW_DELETE, instance.ToJson())

	return nil
}

/**
* newFlowFn
* @param tag, version, name, description string, fn FnContext, stop bool, createdBy string
* @return *Flow
**/
func (s *WorkFlows) newFlowFn(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := newFlowFn(tag, version, name, description, fn, stop, createdBy)
	s.Flows[tag] = flow

	return flow
}

/**
* newFlowDefinition
* @param tag, version, name, description string, definition string, stop bool, createdBy string
* @return *Flow
**/
func (s *WorkFlows) newFlowDefinition(tag, version, name, description string, definition string, stop bool, createdBy string) *Flow {
	flow := newFlowDefinition(tag, version, name, description, definition, stop, createdBy)
	s.Flows[tag] = flow

	return flow
}

/**
* deleteFlow
* @param tag string
* @return error
**/
func (s *WorkFlows) deleteFlow(tag string) error {
	if delFlow != nil {
		delFlow(tag)
	}

	if s.Flows[tag] == nil {
		return nil
	}

	flow := s.Flows[tag]
	event.Publish(EVENT_FLOW_DELETE, flow.ToJson())
	delete(s.Flows, tag)

	return nil
}
