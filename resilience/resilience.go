// Package resilience retries a Go function until it succeeds, up to TotalAttempts, waiting Interval between attempts.
//
// Invariants:
//   - Redis keys: the state of a running instance is in "resilience:instance:<id>" and the stop request in
//     "resilience:<id>:stop"; both expire after the instance's Interval.
//   - Instance.Run is synchronous: it blocks until the function succeeds, the attempts run out or Stop is called.
//   - Instance has no lock; only the instances map of Resilience is protected (mu). An instance is run by one caller.
package resilience

import (
	"encoding/json"
	"errors"
	"reflect"
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
	EVENT_INSTANCE_SET = "resilience:instance:set"
	EVENT_STATUS       = "resilience:status"
)

type Resilience struct {
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	ID        string               `json:"id"`
	instances map[string]*Instance `json:"-"`
	count     int                  `json:"-"`
	mu        sync.Mutex           `json:"-"`
	isDebug   bool                 `json:"-"`
}

/**
* New
* @return *Resilience, error
**/
func New() (*Resilience, error) {
	err := event.Load()
	if err != nil {
		logs.Logf(packageName, MSG_EVENT_NOT_LOADED, err)
	}

	now := timezone.Now()
	result := &Resilience{
		CreatedAt: now,
		UpdatedAt: now,
		ID:        reg.UUID(),
		instances: make(map[string]*Instance),
		mu:        sync.Mutex{},
		isDebug:   envar.GetBool("DEBUG", false),
	}

	return result, nil
}

/**
* ToJson: The resilience as json (ids, dates, running instances and their count).
* @return et.Json
**/
func (s *Resilience) ToJson() et.Json {
	return et.Json{
		"created_at": s.CreatedAt,
		"updated_at": s.UpdatedAt,
		"id":         s.ID,
		"instances":  s.instances,
		"count":      s.count,
	}
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
	s.count++
	go event.Publish(EVENT_STATUS, s.ToJson())
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
	s.count--
}

/**
* newInstance
* @param id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, fn interface{}, fnArgs ...interface{}
* @return Instance
**/
func (s *Resilience) newInstance(id, tag, description string, totalAttempts int, interval time.Duration, tags et.Json, fn interface{}, fnArgs ...interface{}) *Instance {
	id = reg.GetUUID(id)
	now := timezone.Now()
	result := &Instance{
		CreatedAt:     now,
		UpdatedAt:     now,
		ResilienceId:  s.ID,
		ID:            id,
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
		resilience:    s,
	}
	result.setStatus(PENDING)
	s.addInstance(result)

	return result
}

/**
* readInstance
* @param id string
* @return *Instance, bool
**/
func (s *Resilience) readInstance(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	str, exists, err := cache.Get(instanceKey(id), "")
	if err != nil {
		return nil, false
	}

	if !exists || str == "" {
		return nil, false
	}

	bt := []byte(str)
	var result *Instance
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, false
	}

	return result, true
}

type Params struct {
	Id            string
	Tag           string
	Description   string
	TotalAttempts int
	Interval      time.Duration
	Tags          et.Json
	Fn            interface{}
	FnArgs        []interface{}
}

/**
* LoadInstance: The instance with params.Id if it is already in memory, or a new one (3 attempts and 30s by default);
* an id still running in Redis from another process is an error.
* @param params Params
* @return *Instance, error
**/
func (s *Resilience) LoadInstance(params Params) (*Instance, error) {
	instance, exist := s.getInstance(params.Id)
	if exist {
		return instance, nil
	}

	_, exist = s.readInstance(params.Id)
	if exist {
		return nil, errors.New(MSG_INSTANCE_IS_RUNNING)
	}

	if params.TotalAttempts <= 0 {
		params.TotalAttempts = 3
	}

	if params.Interval <= 0 {
		params.Interval = 30 * time.Second
	}

	params.Id = reg.GetUUID(params.Id)
	return s.newInstance(params.Id, params.Tag, params.Description, params.TotalAttempts, params.Interval, params.Tags, params.Fn, params.FnArgs...), nil
}

/**
* Stop
* @param id string
* @return error
**/
func (s *Resilience) Stop(id string) error {
	instance, exist := s.getInstance(id)
	if !exist {
		instance, exist = s.readInstance(id)
		if !exist {
			return errors.New(MSG_ID_NOT_FOUND)
		}
	} else if instance != nil {
		instance.setStop()
	}

	if instance == nil {
		return errors.New(MSG_ID_NOT_FOUND)
	}

	cache.Set(stopKey(id), true, instance.Interval)
	return nil
}
