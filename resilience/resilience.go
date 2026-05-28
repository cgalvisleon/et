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
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
)

type Resilience struct {
	instances map[string]*Instance `json:"-"`
	mu        sync.Mutex           `json:"-"`
	store     instances.Store      `json:"-"`
	isDebug   bool                 `json:"-"`
}

/**
* New
* @return *Resilience, error
 */
func New(store instances.Store) (*Resilience, error) {
	err := event.Load()
	if err != nil {
		return nil, err
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
* add
* @param instance *Instance
* @return void
 */
func (s *Resilience) add(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.instances[instance.ID] = instance
}

/**
* get
* @param id string
* @return *Instance, bool
 */
func (s *Resilience) get(id string) (*Instance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[id]
	return instance, ok
}

/**
* remove
* @param id string
* @return void
 */
func (s *Resilience) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.instances, id)
}

/**
* Count
* @return int
 */
func (s *Resilience) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.instances)
}

/**
* new
* @param tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return Instance
 */
func (s *Resilience) new(tag, description, ownerId string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	id := reg.UUID()
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
	result.setStatus(StatusPending)
	s.add(result)

	return result
}

/**
* Get
* @param id string
* @return *Instance, bool
**/
func (s *Resilience) Get(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, exist := s.get(id)
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
		s.add(result)

		if s.isDebug {
			logs.Log(packageName, "load:", result.ToString())
		}

		return result, true
	}

	return nil, false
}

/**
* Run
* @param tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func (s *Resilience) Run(tag, description, ownerId string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if totalAttempts <= 0 {
		totalAttempts = 3
	}

	if interval <= 0 {
		interval = 30 * time.Second
	}

	result := s.new(tag, description, ownerId, totalAttempts, interval, tags, team, level, fn, fnArgs...)
	result.Run()

	return result
}

/**
* RunCustom
* @param tag, description, ownerId string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func (s *Resilience) RunCustom(tag, description, ownerId string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	totalAttempts := envar.GetInt("RESILIENCE_TOTAL_ATTEMPTS", 3)
	intervalSeconds := envar.GetInt("RESILIENCE_INTERVAL_SECONDS", 30)
	interval := time.Duration(intervalSeconds) * time.Second

	return s.Run(tag, description, ownerId, totalAttempts, interval, tags, team, level, fn, fnArgs...)
}

/**
* Stop
* @param id string
* @return error
 */
func (s *Resilience) Stop(id string) error {
	result, exist := s.Get(id)
	if !exist {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	result.Stop()

	return nil
}

/**
* Restart
* @param id string
* @return error
 */
func (s *Resilience) Restart(id string) error {
	result, exist := s.Get(id)
	if !exist {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	result.Restart()

	return nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
 */
func (s *Resilience) Query(query et.Json) (et.Items, error) {
	if s.store == nil {
		return et.Items{}, errors.New(msg.MSG_STORE_IS_REQUIRED)
	}

	return s.store.Query(query)
}
