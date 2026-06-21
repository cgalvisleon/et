package jwf

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

const (
	packageName = "workflow"
)

type Store interface {
	Set(collection, id, tenantId, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
	GenSerie(TenantId, tag string) (string, error)
}

type WorkFlow struct {
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	TenantId  string           `json:"tenant_id"`
	ID        string           `json:"id"`
	AuditLog  []et.Json        `json:"audit_log"`
	Flows     map[string]*Flow `json:"-"`
	Steps     map[string]*Step `json:"-"`
	bindings  map[string]any   `json:"-"`
	muFlows   sync.Mutex       `json:"-"`
	muSteps   sync.Mutex       `json:"-"`
	store     Store            `json:"-"`
	isDebug   bool             `json:"-"`
	isChanged bool             `json:"-"`
}

/**
* New
* @param tenantId string, store Store
* @return *WorkFlow
**/
func New(tenantId string, store Store) (*WorkFlow, error) {
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	err = event.Load()
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	result := &WorkFlow{
		CreatedAt: now,
		UpdatedAt: now,
		TenantId:  tenantId,
		ID:        reg.ULID(),
		Flows:     make(map[string]*Flow),
		Steps:     make(map[string]*Step),
		AuditLog:  make([]et.Json, 0),
	}
	return result.up(store)
}

/**
* Load
* @param tenantId string, store Store
* @return *WorkFlow, error
**/
func Load(id string, store Store) (*WorkFlow, error) {
	if store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	var def et.Json
	exists, err := store.Get("workflow", id, &def)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_WORKFLOW_NOT_FOUND)
	}

	result := &WorkFlow{}
	err = json.Unmarshal([]byte(def.ToString()), &result)
	if err != nil {
		return nil, err
	}

	return result.up(store)
}

/**
* up
* @param store Store
* @return *WorkFlow
**/
func (s *WorkFlow) up(store Store) (*WorkFlow, error) {
	isDebug := envar.GetBool("DEBUG", false)
	s.bindings = make(map[string]any)
	s.muFlows = sync.Mutex{}
	s.muSteps = sync.Mutex{}
	s.store = store
	s.isDebug = isDebug
	for id := range s.Flows {
		_, err := s.loadFlow(id)
		if err != nil {
			return nil, err
		}
	}
	for id := range s.Steps {
		_, err := s.loadStep(id)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *WorkFlow) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
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
}

/**
* ToJson
* @return et.Json
**/
func (s *WorkFlow) ToJson() et.Json {
	flows := et.Json{}
	for id, flow := range s.Flows {
		flows[id] = flow.ToJson()
	}
	steps := et.Json{}
	for id, step := range s.Steps {
		steps[id] = step.ToJson()
	}
	return et.Json{
		"created_at": timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":  s.TenantId,
		"id":         s.ID,
		"flows":      flows,
		"steps":      steps,
		"audit_log":  s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *WorkFlow) ToString() string {
	return s.ToJson().ToString()
}

/**
* Save
* @return error
**/
func (s *WorkFlow) Save() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false
	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	return s.store.Set("workflow", s.ID, s.TenantId, s.TenantId, s)
}

/**
* Delete
* @return error
**/
func (s *WorkFlow) Delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	return s.store.Delete("workflow", s.ID)
}

/**
* SetBinding
* @param key string, value any
**/
func (s *WorkFlow) SetBinding(key string, value any) {
	s.bindings[key] = value
}

/**
* addFlow
* @param flow *Flow
**/
func (s *WorkFlow) addFlow(flow *Flow) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	s.Flows[flow.ID] = flow
}

/**
* addStep
* @param step *Step
**/
func (s *WorkFlow) getFlow(id string) (*Flow, bool) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	flow, exists := s.Flows[id]
	if !exists {
		return nil, false
	}

	return flow, true
}

/**
* removeFlow
* @param id string
**/
func (s *WorkFlow) removeFlow(id string) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	delete(s.Flows, id)
}

/**
* addInstance
* @param instance *Instance
**/
func (s *WorkFlow) addStep(step *Step) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	s.Steps[step.ID] = step
}

/**
* getStep
* @param id string
* @return *Step, bool
**/
func (s *WorkFlow) getStep(id string) (*Step, bool) {
	s.muSteps.Lock()
	defer s.muSteps.Unlock()

	step, exists := s.Steps[id]
	if !exists {
		return nil, false
	}

	return step, true
}

/**
* removeStep
* @param id string
**/
func (s *WorkFlow) removeStep(id string) {
	s.muSteps.Lock()
	defer s.muSteps.Unlock()

	delete(s.Steps, id)
}

/**
* NewFlow
* @param tag, title, version string
* @return *Flow
**/
func (s *WorkFlow) NewFloW(tag, title, version, userId string) *Flow {
	result := s.newFlow(tag, title, version, userId)
	s.addFlow(result)
	return result
}

/**
* Run
* @param tag, id, ownerId string, step int, ctx, tags et.Json, userId string
* @return *Instance, error
**/
func (s *WorkFlow) Run(flowId, triggerTag, id, projectId string, ctx, tags et.Json, userId string) (et.Json, error) {
	id = reg.GetULID(id)
	instance, err := s.getInstance(id, userId)
	if errors.Is(err, ErrorInstanceNotFound) {
		instance, err = s.newInstance(projectId, flowId, triggerTag, userId)
		if err != nil {
			return nil, err
		}
		instance.setStatus(PENDING, userId)
	}
	if err != nil {
		return nil, err
	}

	instance.setTag(tags)
	instance.setCtx(ctx)
	result, err := instance.run(ctx, userId)
	if err != nil {
		return et.Json{}, err
	}

	key := fmt.Sprintf("instance:%s:status", id)
	cache.Delete(key)

	return result, nil
}
