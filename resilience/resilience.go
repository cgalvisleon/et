package resilience

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
)

type Resilience struct {
	instances map[string]*Instance `json:"-"`
	mu        sync.Mutex           `json:"-"`
	store     jsql.Store           `json:"-"`
	isDebug   bool                 `json:"-"`
}

/**
* New
* @param store jsql.Store
* @return *Resilience, error
**/
func New(store jsql.Store) (*Resilience, error) {
	err := event.Load()
	if err != nil {
		logs.Logf(packageName, MSG_EVENT_NOT_LOADED, err)
	}

	result := &Resilience{
		instances: make(map[string]*Instance),
		mu:        sync.Mutex{},
		isDebug:   envar.GetBool("DEBUG", false),
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
* @param id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return Instance
**/
func (s *Resilience) newInstance(id, tag, description, ownerId string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if id == "" {
		id = reg.ULID()
	}
	if ownerId == "" {
		ownerId = id
	}
	result := &Instance{
		CreatedAt:     time.Now(),
		ID:            id,
		Tag:           tag,
		OwnerId:       ownerId,
		Description:   description,
		fn:            fn,
		fnArgs:        fnArgs,
		fnResult:      []reflect.Value{},
		TotalAttempts: totalAttempts,
		Interval:      interval,
		Tags:          tags,
		Team:          team,
		Level:         level,
		stop:          false,
	}
	result.setStatus(PENDING)
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

/**
* RunInstance
* @param id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
**/
func (s *Resilience) LoadInstance(id, tag, description, ownerId string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if totalAttempts <= 0 {
		totalAttempts = 3
	}

	if interval <= 0 {
		interval = 30 * time.Second
	}

	id = reg.GetULID(id)
	result, exist := s.GetInstance(id)
	if !exist {
		result = s.newInstance(id, tag, description, ownerId, totalAttempts, interval, tags, team, level, fn, fnArgs...)
	}

	return result
}

/**
* Stop
* @param id string
* @return error
**/
func (s *Resilience) Stop(id string) error {
	result, exist := s.GetInstance(id)
	if !exist {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	result.setStop()

	return nil
}

/**
* Restart
* @param id string
* @return error
**/
func (s *Resilience) Restart(id string) error {
	result, exist := s.GetInstance(id)
	if !exist {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	result.setRestart()

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
