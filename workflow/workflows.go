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
)

var (
	packageName           = "workflow"
	ErrorInstanceNotFound = fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
)

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error

type WorkFlow struct {
	Flows       map[string]*Flow       `json:"flows"`
	Instances   map[string]*Instance   `json:"instances"`
	Results     map[string]et.Json     `json:"results"`
	mu          sync.Mutex             `json:"-"`
	getInstance GetInstanceFn          `json:"-"`
	setInstance SetInstanceFn          `json:"-"`
	resilience  *resilience.Resilience `json:"-"`
	isDebug     bool                   `json:"-"`
}

/**
* add
* @param instance *Instance
**/
func (s *WorkFlow) add(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Instances[instance.ID] = instance
}

/**
* Get
* @param id string
* @return *Instance, bool
**/
func (s *WorkFlow) Get(id string) (*Instance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.Instances[id]
	return instance, ok
}

/**
* remove
* @param instance *Instance
**/
func (s *WorkFlow) remove(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Instances, instance.ID)
}

/**
* Count
* @return int
**/
func (s *WorkFlow) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.Instances)
}

/**
* new
* @param tag, id string, tags et.Json, step int, createdBy string
* @return *Instance, error
**/
func (s *WorkFlow) new(tag, id string, tags et.Json, step int, createdBy string) (*Instance, error) {
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
		owner:      s,
		Tag:        tag,
		CreatedAt:  now,
		UpdatedAt:  now,
		ID:         id,
		CreatedBy:  createdBy,
		UpdatedBy:  createdBy,
		Current:    step,
		Ctx:        et.Json{},
		Ctxs:       make(map[int]et.Json),
		Results:    make(map[int]*Result),
		Rollbacks:  make(map[int]*Result),
		Tags:       tags,
		Resilence:  make(map[string]*resilience.Instance),
		goTo:       -1,
		WorkerHost: workerHost,
		Params:     et.Json{},
		isNew:      true,
	}
	result.setStatus(Pending)
	s.add(result)

	return result, nil
}

/**
* load
* @param id string
* @return *Flow, error
**/
func (s *WorkFlow) load(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, exists := s.Get(id)
	if exists {
		return result, true
	}

	if s.getInstance != nil {
		exists, err := s.getInstance(id, &result)
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
		s.add(result)

		if s.isDebug {
			logs.Log(packageName, "load:", result.ToString())
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
func (s *WorkFlow) getOrCreateInstance(id, tag string, step int, tags et.Json, createdBy string) (*Instance, error) {
	id = reg.GetUUID(id)
	result, exists := s.load(id)
	if !exists {
		return s.new(tag, id, tags, step, createdBy)
	}

	return result, nil
}

/**
* runInstance
* Si el step es -1 se ejecuta el siguiente paso, si no se ejecuta el paso indicado
* @param instanceId, tag string, step int, ctx, tags et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlow) runInstance(instanceId, tag string, step int, ctx, tags et.Json, createdBy string) (et.Json, error) {
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

	s.remove(instance)
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
func (s *WorkFlow) resetInstance(instanceId, updatedBy string) error {
	instance, exists := s.load(instanceId)
	if !exists {
		return fmt.Errorf("instance not found")
	}

	instance.UpdatedBy = updatedBy
	instance.Current = 0
	instance.setStatus(Pending)

	return nil
}

/**
* Rollback
* @param instanceId string, updatedBy string
* @return et.Json, error
**/
func (s *WorkFlow) rollback(instanceId, updatedBy string) (et.Json, error) {
	instance, exists := s.load(instanceId)
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
func (s *WorkFlow) stop(instanceId, updatedBy string) error {
	instance, exists := s.load(instanceId)
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
func (s *WorkFlow) newFlow(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := newFlow(tag, version, name, description, fn, stop, createdBy)
	s.Flows[tag] = flow

	return flow
}

/**
* deleteFlow
* @param tag string
* @return bool
**/
func (s *WorkFlow) deleteFlow(tag string) bool {
	if s.Flows[tag] == nil {
		return false
	}

	flow := s.Flows[tag]
	event.Publish(EVENT_WORKFLOW_DELETE, flow.ToJson())
	delete(s.Flows, tag)

	return true
}

/**
* New
* @param tag, version, name, description string, fn FnContext, createdBy string
* @return *Flow
**/
func (s *WorkFlow) New(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	return s.newFlow(tag, version, name, description, fn, stop, createdBy)
}

/**
* Run
* @param instanceId, tag string, step int, ctx, tags et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlow) Run(instanceId, tag string, step int, ctx, tags et.Json, createdBy string) (et.Json, error) {
	return s.runInstance(instanceId, tag, step, ctx, tags, createdBy)
}

/**
* Reset
* @param instanceId, updatedBy string
* @return error
**/
func (s *WorkFlow) Reset(instanceId, updatedBy string) error {
	return s.resetInstance(instanceId, updatedBy)
}

/**
* Rollback
* @param instanceId, updatedBy string
* @return et.Json, error
**/
func (s *WorkFlow) Rollback(instanceId, updatedBy string) (et.Json, error) {
	return s.rollback(instanceId, updatedBy)
}

/**
* Stop
* @param instanceId, updatedBy string
* @return error
**/
func (s *WorkFlow) Stop(instanceId, updatedBy string) error {
	return s.stop(instanceId, updatedBy)
}

/**
* SetStatus
* @param instanceId, status, updatedBy string
* @return FlowStatus, error
**/
func (s *WorkFlow) Status(instanceId, status, updatedBy string) (Status, error) {
	if _, ok := FlowStatusList[Status(status)]; !ok {
		return "", fmt.Errorf("status %s no es valido", status)
	}

	instance, exists := s.load(instanceId)
	if !exists {
		return "", fmt.Errorf("instance not found")
	}

	instance.setStatus(Status(status))
	return instance.Status, nil
}

/**
* DeleteFlow
* @param tag string
* @return (bool, error)
**/
func (s *WorkFlow) DeleteFlow(tag string) (bool, error) {
	return s.deleteFlow(tag), nil
}

/**
* GetInstance
* @param instanceId string
* @return (*Instance, error)
**/
func (s *WorkFlow) GetInstance(instanceId string) (*Instance, error) {
	instance, exists := s.load(instanceId)
	if !exists {
		return nil, fmt.Errorf("instance not found")
	}

	return instance, nil
}
