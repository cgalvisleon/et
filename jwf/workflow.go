package jwf

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

const (
	packageName = "workflow"
)

type Store interface {
	Set(collection, id, tenantId, ownerId string, obj any, userId string) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
	GetCode(tag string) (string, error)
}

type WorkFlow struct {
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	TenantId        string               `json:"tenant_id"`
	ID              string               `json:"id"`
	Flows           map[string]*Flow     `json:""`
	Instances       map[string]*Instance `json:"-"`
	AuditLog        []et.Json            `json:"audit_log"`
	bindings        map[string]any       `json:"-"`
	muFlows         sync.Mutex           `json:"-"`
	muInstances     sync.Mutex           `json:"-"`
	store           Store                `json:"-"`
	resilienceStore resilience.Store     `json:"-"`
	metrics         cache.Metrics        `json:"-"`
	isDebug         bool                 `json:"-"`
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

	isDebug := config.GetBool("DEBUG", false)
	now := timezone.Now()
	id := fmt.Sprintf("workflow:%s", tenantId)
	return &WorkFlow{
		CreatedAt:   now,
		UpdatedAt:   now,
		TenantId:    tenantId,
		ID:          id,
		Flows:       make(map[string]*Flow),
		Instances:   make(map[string]*Instance),
		AuditLog:    make([]et.Json, 0),
		bindings:    make(map[string]any),
		muFlows:     sync.Mutex{},
		muInstances: sync.Mutex{},
		store:       store,
		metrics:     cache.Metrics{},
		isDebug:     isDebug,
	}, nil
}

/**
* Load
* @param store Store
* @return *WorkFlow, error
**/
func Load(tenantId string, store Store, userId string) (*WorkFlow, error) {
	if store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	id := fmt.Sprintf("workflow:%s", tenantId)
	result := &WorkFlow{}
	exists, err := store.Get("workflow", id, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		result, err = New(tenantId, store)
		if err != nil {
			return nil, err
		}

		err = result.Save(userId)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	isDebug := config.GetBool("DEBUG", false)
	result.store = store
	result.metrics = cache.Metrics{}
	result.isDebug = isDebug
	return result, nil
}

/**
* Save
* @return error
**/
func (s *WorkFlow) Save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     "save",
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}

	return s.store.Set("workflow", s.ID, s.TenantId, s.TenantId, s, userId)
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
func (s *WorkFlow) addInstance(instance *Instance) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	s.Instances[instance.ID] = instance
}

/**
* removeInstance
* @param id string
**/
func (s *WorkFlow) removeInstance(id string) {
	s.muInstances.Lock()
	defer s.muInstances.Unlock()

	key := fmt.Sprintf("%s:status", id)
	cache.Delete(key)
	delete(s.Instances, id)
}

/**
* NewFlow
* @param tag, title, version string
* @return *Flow
**/
func (s *WorkFlow) NewFloW(tag, title, version string) *Flow {
	return s.newFlow(tag, title, version)
}

/**
* Run
* @param tag, id, ownerId string, step int, ctx, tags et.Json, userId string
* @return *Instance, error
**/
func (s *WorkFlow) Run(flowId, triggerTag, id, projectId string, ctx, tags et.Json, userId string) (et.Json, error) {
	if id != "" {
		key := fmt.Sprintf("%s:status", id)
		exists := cache.Exists(key)
		if exists {
			status, err := cache.Get(key, string(PENDING))
			if err != nil {
				return et.Json{}, err
			}
			return et.Json{}, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, status)
		}
		cache.Set(key, PENDING, 3*time.Second)
	}

	instance, err := s.getInstance(id, userId)
	if errors.Is(err, ErrorInstanceNotFound) {
		instance, err = s.newInstance(projectId, flowId, triggerTag, userId)
		instance.setStatus(PENDING, userId)
	}
	if err != nil {
		return nil, err
	}

	s.addInstance(instance)
	instance.setTag(tags)
	instance.setCtx(ctx)
	result, err := instance.run(ctx, userId)
	if err != nil {
		return et.Json{}, err
	}

	s.removeInstance(instance.ID)

	return result, nil
}
