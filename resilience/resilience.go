package resilience

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
)

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error

type Resilience struct {
	instances   map[string]*Instance `json:"-"`
	mu          sync.Mutex           `json:"-"`
	getInstance GetInstanceFn        `json:"-"`
	setInstance SetInstanceFn        `json:"-"`
	isDebug     bool                 `json:"-"`
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
* Get
* @param id string
* @return *Instance, bool
 */
func (s *Resilience) Get(id string) (*Instance, bool) {
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
func (s *Resilience) new(tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	id := reg.UUID()
	result := &Instance{
		CreatedAt:     time.Now(),
		ID:            id,
		Tag:           tag,
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
* Load
* @param id string
* @return *Instance, bool
**/
func (s *Resilience) load(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, exist := s.Get(id)
	if exist {
		return result, true
	}

	if s.getInstance != nil {
		exist, err := s.getInstance(id, &result)
		if err != nil {
			return nil, false
		}

		if !exist {
			return nil, false
		}

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
func (s *Resilience) Run(tag, description string, totalAttempts int, interval time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if totalAttempts <= 0 {
		totalAttempts = 3
	}

	if interval <= 0 {
		interval = 30 * time.Second
	}

	result := s.new(tag, description, totalAttempts, interval, tags, team, level, fn, fnArgs...)
	result.Run()

	return result
}

/**
* RunCustom
* @param tag, description string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func (s *Resilience) RunCustom(tag, description string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	totalAttempts := envar.GetInt("RESILIENCE_TOTAL_ATTEMPTS", 3)
	intervalSeconds := envar.GetInt("RESILIENCE_INTERVAL_SECONDS", 30)
	interval := time.Duration(intervalSeconds) * time.Second

	return s.Run(tag, description, totalAttempts, interval, tags, team, level, fn, fnArgs...)
}

/**
* Stop
* @param id string
* @return error
 */
func (s *Resilience) Stop(id string) error {
	result, exist := s.load(id)
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
	result, exist := s.load(id)
	if !exist {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	result.Restart()

	return nil
}
