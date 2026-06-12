package resilience

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
)

type Store interface {
	Set(tag, id, tenantId, ownerId string, obj any, userId string) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	Query(query et.Json) (et.Items, error)
}

type Resilience struct {
	instances map[string]*Instance `json:"-"`
	mu        sync.Mutex           `json:"-"`
	store     Store                `json:"-"`
	metrics   cache.Metrics        `json:"-"`
	isDebug   bool                 `json:"-"`
}

/**
* New
* @param store Store
* @return *Resilience, error
**/
func New(store Store) (*Resilience, error) {
	err := event.Load()
	if err != nil {
		logs.Logf(packageName, MSG_EVENT_NOT_LOADED, err)
	}

	result := &Resilience{
		instances: make(map[string]*Instance),
		mu:        sync.Mutex{},
		isDebug:   config.GetBool("DEBUG", false),
		store:     store,
	}

	return result, nil
}

/**
* addInstance
* @param instance *Instance
* @return void
**/
func (s *Resilience) addInstance(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.instances[instance.ID] = instance
}

/**
* getInstance
* @param id string
* @return *Instance, bool
**/
func (s *Resilience) getInstance(id string) (*Instance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[id]
	return instance, ok
}

/**
* removeInstance
* @param id string
**/
func (s *Resilience) removeInstance(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.instances, id)
}

/**
* CountInstances
* @return int
**/
func (s *Resilience) CountInstances() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.instances)
}

/**
* newInstance
* @param id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, userId string, fn interface{}, fnArgs ...interface{}
* @return Instance
**/
func (s *Resilience) newInstance(tenantId, id, tag, description, ownerId string, totalAttempts int, interval time.Duration, tags et.Json, userId string, fn interface{}, fnArgs ...interface{}) *Instance {
	if id == "" {
		id = reg.ULID()
	}
	if ownerId == "" {
		ownerId = id
	}

	result := &Instance{
		CreatedAt:     time.Now(),
		ID:            id,
		TenantId:      tenantId,
		OwnerId:       ownerId,
		Tag:           tag,
		Description:   description,
		fn:            fn,
		fnArgs:        fnArgs,
		fnResult:      []reflect.Value{},
		TotalAttempts: totalAttempts,
		Interval:      interval,
		Tags:          tags,
		Result:        make([]any, 0),
		stop:          false,
	}
	result.setStatus(PENDING, userId)
	s.addInstance(result)

	return result
}

/**
* GetInstance
* @param id string
* @return *Instance, bool
**/
func (s *Resilience) GetInstance(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, exist := s.getInstance(id)
	if exist {
		return result, true
	}

	if s.store != nil {
		exist, err := s.store.Get(id, &result)
		if err != nil {
			return nil, false
		}

		if !exist {
			return nil, false
		}

		result.up(s)
		s.addInstance(result)

		if s.isDebug {
			logs.Log(packageName, "load:", result.ToString())
		}

		return result, true
	}

	return nil, false
}

type Params struct {
	TenantId      string
	Id            string
	Tag           string
	Description   string
	OwnerId       string
	TotalAttempts int
	Interval      time.Duration
	Tags          et.Json
	UserId        string
	Fn            interface{}
	FnArgs        []interface{}
}

/**
* RunInstance
* @param tenantId, id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team, level, userId string, fn interface{}, fnArgs ...interface{}
* @return *Instance
**/
func (s *Resilience) LoadInstance(params Params) *Instance {
	if params.TotalAttempts <= 0 {
		params.TotalAttempts = 3
	}

	if params.Interval <= 0 {
		params.Interval = 30 * time.Second
	}

	params.Id = reg.GetULID(params.Id)
	result, exist := s.GetInstance(params.Id)
	if !exist {
		result = s.newInstance(params.TenantId, params.Id, params.Tag, params.Description, params.OwnerId, params.TotalAttempts, params.Interval, params.Tags, params.UserId, params.Fn, params.FnArgs...)
	}

	return result
}

/**
* Stop
* @param id string
* @return error
**/
func (s *Resilience) Stop(id, userId string) error {
	result, exist := s.GetInstance(id)
	if !exist {
		return errors.New(MSG_ID_NOT_FOUND)
	}

	result.setStop(userId)

	return nil
}

/**
* Restart
* @param id string
* @return error
**/
func (s *Resilience) Restart(id, userId string) error {
	result, exist := s.GetInstance(id)
	if !exist {
		return errors.New(MSG_ID_NOT_FOUND)
	}

	result.setRestart(userId)

	return nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Resilience) Query(query et.Json) (et.Items, error) {
	if s.store == nil {
		return et.Items{}, errors.New(msg.MSG_STORE_IS_REQUIRED)
	}

	return s.store.Query(query)
}
