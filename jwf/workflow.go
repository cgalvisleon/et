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

type WorkFlow struct {
	CreatedAt time.Time                        `json:"created_at"`
	UpdatedAt time.Time                        `json:"updated_at"`
	ID        string                           `json:"id"`
	AuditLog  []et.Json                        `json:"audit_log"`
	Steps     map[string]*Step                 `json:"steps"`
	Flows     map[string]*Flow                 `json:"flows"`
	bindings  map[string]any                   `json:"-"`
	muFlows   sync.Mutex                       `json:"-"`
	muSteps   sync.Mutex                       `json:"-"`
	isInitial bool                             `json:"-"`
	store     Store                            `json:"-"`
	isDebug   bool                             `json:"-"`
	isChanged bool                             `json:"-"`
	onSave    []func(workflow *WorkFlow) error `json:"-"`
	onDelete  []func(workflow *WorkFlow) error `json:"-"`
}

/**
* New
* @param store Store
* @return *WorkFlow
**/
func New(store Store, userID string) (*WorkFlow, error) {
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
		ID:        reg.UUID(),
		AuditLog:  make([]et.Json, 0),
		Steps:     make(map[string]*Step),
		Flows:     make(map[string]*Flow),
		muFlows:   sync.Mutex{},
		muSteps:   sync.Mutex{},
		store:     store,
	}
	result.addAuditLog(userID, "new_workflow")
	_, err = result.up()
	if err != nil {
		return nil, err
	}

	err = result.Save()
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* Load
* @param store Store, id string
* @return *WorkFlow, error
**/
func Load(store Store, id string) (*WorkFlow, error) {
	if store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	var def et.Json
	exists, err := store.Get("workflows", id, &def)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_WORKFLOW_NOT_FOUND)
	}

	result := &WorkFlow{
		ID:    id,
		store: store,
	}
	result.CreatedAt = def.Time("created_at")
	result.UpdatedAt = def.Time("updated_at")
	result.AuditLog = def.ArrayJson("audit_log")
	result.Steps = make(map[string]*Step)
	result.Flows = make(map[string]*Flow)
	result.muFlows = sync.Mutex{}
	result.muSteps = sync.Mutex{}

	steps := def.Json("steps")
	for id := range steps {
		_, err := result.loadStep(id)
		if err != nil {
			return nil, err
		}
	}

	flows := def.Json("flows")
	for id := range flows {
		_, err := result.loadFlow(id)
		if err != nil {
			return nil, err
		}
	}

	_, err = result.up()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* up
* @param store Store
* @return *WorkFlow
**/
func (s *WorkFlow) up() (*WorkFlow, error) {
	isDebug := envar.GetBool("DEBUG", false)
	s.bindings = make(map[string]any)
	s.isDebug = isDebug
	s.onSave = make([]func(workflow *WorkFlow) error, 0)
	s.onDelete = make([]func(workflow *WorkFlow) error, 0)
	s.OnSave(func(workflow *WorkFlow) error {
		key := fmt.Sprintf("workflow:%s", workflow.ID)
		event.Publish(key, workflow.ToJson())
		return nil
	})
	s.OnDelete(func(workflow *WorkFlow) error {
		key := fmt.Sprintf("workflow:%s:delete", workflow.ID)
		event.Publish(key, et.Json{
			"id": workflow.ID,
		})
		return nil
	})
	err := s.init()
	if err != nil {
		return nil, err
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
}

/**
* Ref
* @return et.Json
**/
func (s *WorkFlow) Ref() et.Json {
	flows := et.Json{}
	for id, flow := range s.Flows {
		flows[id] = flow.ref()
	}
	steps := et.Json{}
	for id, step := range s.Steps {
		steps[id] = step.ref()
	}
	return et.Json{
		"id":    s.ID,
		"flows": flows,
		"steps": steps,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *WorkFlow) ToJson() et.Json {
	return et.Json{
		"created_at": timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"id":         s.ID,
		"flows":      s.Flows,
		"steps":      s.Steps,
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
* OnSave
* @param fn func(workflow *WorkFlow) error
* @return *WorkFlow
**/
func (s *WorkFlow) OnSave(fn func(workflow *WorkFlow) error) *WorkFlow {
	if s.onSave == nil {
		s.onSave = make([]func(workflow *WorkFlow) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(workflow *WorkFlow) error
* @return *WorkFlow
**/
func (s *WorkFlow) OnDelete(fn func(workflow *WorkFlow) error) *WorkFlow {
	if s.onDelete == nil {
		s.onDelete = make([]func(workflow *WorkFlow) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
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

	err := s.store.Set("workflows", s.ID, s.ID, s.Ref())
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		if err := onSave(s); err != nil {
			return err
		}
	}

	return nil
}

/**
* Delete
* @return error
**/
func (s *WorkFlow) Delete() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	err := s.store.Delete("workflow", s.ID)
	if err != nil {
		return err
	}

	for _, onDelete := range s.onDelete {
		if err := onDelete(s); err != nil {
			return err
		}
	}

	return nil
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
func (s *WorkFlow) removeStep(id, userId string) {
	s.muSteps.Lock()
	defer s.muSteps.Unlock()

	s.addAuditLog(userId, "remove_step")
	delete(s.Steps, id)
}

/**
* NewFlow
* @param tag, title, version string
* @return *Flow
**/
func (s *WorkFlow) NewFloW(tag, title, version, userId string) *Flow {
	result := s.newFlow(tag, title, version, userId)
	s.addAuditLog(userId, "new_flow")
	s.addFlow(result)
	return result
}

/**
* SetStep
* @param stepDef et.Json
* @return *Flow, error
**/
func (s *WorkFlow) SetStep(stepDef et.Json, userId string) (*WorkFlow, error) {
	var step *Step
	bt, err := stepDef.ToByte()
	if err != nil {
		return s, err
	}

	err = json.Unmarshal(bt, &step)
	if err != nil {
		return s, err
	}

	if step.ID == "" {
		step.ID = reg.UUID()
	}
	step.TypeId = step.ID
	step.OwnerId = s.ID
	step.up(s)
	s.addAuditLog(userId, "new_step")
	s.addStep(step)
	return s, nil
}

/**
* Run
* @param flowId, triggerTag, id, projectId, code string, ctx, tags et.Json, userId string
* @return *Instance, error
**/
func (s *WorkFlow) Run(flowId, triggerTag, id, projectId, code string, ctx, tags et.Json, userId string) (et.Json, error) {
	id = reg.GetULID(id)
	instance, err := s.getInstance(id, userId)
	if errors.Is(err, ErrorInstanceNotFound) {
		instance, err = s.newInstance(projectId, flowId, triggerTag, code, userId)
		if err != nil {
			return nil, err
		}
		instance.setStatus(PENDING)
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
