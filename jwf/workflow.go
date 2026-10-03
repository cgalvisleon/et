package jwf

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

const (
	EVENT_FLOW_SET = "workflow:flow:set"
	packageName    = "workflow"
)

type WorkFlow struct {
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
	ID        string                     `json:"id"`
	Flows     map[string]*Flow           `json:"flows"`
	TimeAwait time.Duration              `json:"time_await"`
	AuditLog  []et.Json                  `json:"audit_log"`
	bindings  map[string]any             `json:"-"`
	muFlows   sync.Mutex                 `json:"-"`
	isInitial bool                       `json:"-"`
	isDebug   bool                       `json:"-"`
	isChanged bool                       `json:"-"`
	onChange  []func(data et.Json) error `json:"-"`
}

/**
* New
* @param db *jsql.DB, id, userID string
* @return *WorkFlow
**/
func New(id string) (*WorkFlow, error) {
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	err = event.Load()
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	id = reg.GetUUID(id)
	result := &WorkFlow{
		CreatedAt: now,
		UpdatedAt: now,
		ID:        id,
		Flows:     make(map[string]*Flow),
		TimeAwait: 10 * time.Minute,
		AuditLog:  make([]et.Json, 0),
		bindings:  make(map[string]any),
		muFlows:   sync.Mutex{},
		onChange:  make([]func(data et.Json) error, 0),
	}
	result.up()
	return result, nil
}

/**
* up
* @param store Store
* @return *WorkFlow
**/
func (s *WorkFlow) up() *WorkFlow {
	s.isDebug = envar.GetBool("DEBUG", false)
	if s.bindings == nil {
		s.bindings = make(map[string]any)
	}
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	return s
}

/**
* push
* @return error
**/
func (s *WorkFlow) push() {
	for _, onChange := range s.onChange {
		err := onChange(s.ToJson())
		if err != nil {
			return
		}
	}
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
	s.push()
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
* OnChange
* @param fn func(data et.Json) error
* @return *WorkFlow
**/
func (s *WorkFlow) OnChange(fn func(data et.Json) error) *WorkFlow {
	if s.onChange == nil {
		s.onChange = make([]func(data et.Json) error, 0)
	}
	s.onChange = append(s.onChange, fn)
	return s
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

	s.Flows[flow.Tag] = flow
}

/**
* getFlow
* @param tag string
* @return *Flow, bool
**/
func (s *WorkFlow) getFlow(tag string) (*Flow, bool) {
	s.muFlows.Lock()
	defer s.muFlows.Unlock()

	flow, exists := s.Flows[tag]
	if !exists {
		return nil, false
	}

	return flow, true
}

/**
* SetTimeAwait
* @param time time.Duration
* @return *WorkFlow
**/
func (s *WorkFlow) SetTimeAwait(time time.Duration, userId string) *WorkFlow {
	s.TimeAwait = time
	s.addAuditLog(userId, "set_time_await")
	return s
}

/**
* NewFlow
* @param tag, name, version, ownerId, userId string
* @return *Flow
**/
func (s *WorkFlow) NewFlow(tag, name, version, ownerId, userId string) *Flow {
	result := NewFlow(tag, name, version, ownerId, userId)
	s.addAuditLog(userId, "new_flow")
	s.addFlow(result)
	return result
}

/**
* getInstance
* @param id string
* @return *Instance, error
**/
func (s *WorkFlow) getInstance(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	key := fmt.Sprintf("instance:%s", id)
	var def et.Json
	exists, err := cache.GetObject(key, &def)
	if err != nil {
		return nil, false
	}

	if !exists {
		return nil, false
	}

	if def.IsEmpty() {
		return nil, false
	}

	flowTag := def.ValStr("", "flow_tag")
	if flowTag == "" {
		return nil, false
	}

	flow, exists := s.getFlow(flowTag)
	if !exists {
		return nil, false
	}

	result, err := flow.LoadInstance(def)
	if err != nil {
		return nil, false
	}

	result.OnChange(func(data et.Json) error {
		_, err := cache.SetObject(key, data, 0)
		return err
	})

	return result, true
}

/**
* newInstance
* @param params InstanceParams
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(tag, triggerTag, id, code string, tags et.Json, userId string) (*Instance, error) {
	flow, exists := s.getFlow(tag)
	if !exists {
		return nil, ErrrFlowNotFound
	}

	trigger, exists := flow.getTrigger(triggerTag)
	if !exists {
		return nil, ErrrTriggerNotFound
	}

	name := flow.Name
	if code != "" {
		name = fmt.Sprintf("%s %s", flow.Name, code)
	}

	id = reg.GetUUID(id)
	key := fmt.Sprintf("instance:%s", id)
	result := flow.NewInstance(id, code, name, tags, trigger, userId)

	result.OnChange(func(data et.Json) error {
		_, err := cache.SetObject(key, data, 0)
		return err
	})

	s.addAuditLog(userId, "new_instance")
	return result, nil
}

/**
* GetFlow
* @param tag, triggerTag, id, projectId, code, userId string
* @return *Instance, error
**/
func (s *WorkFlow) GetInstance(tag, triggerTag, id, code string, tags et.Json, userId string) (*Instance, error) {
	id = reg.GetULID(id)
	instance, exists := s.getInstance(id)
	if !exists {
		instance, err := s.newInstance(tag, triggerTag, id, code, tags, userId)
		if err != nil {
			return nil, err
		}
		return instance, nil
	}

	return instance, nil
}

/**
* StopInstance
* @param id string
**/
func (s *WorkFlow) StopInstance(id string) {
	key := fmt.Sprintf("instance:%s:stop", id)
	cache.Set(key, "true", s.TimeAwait)
}

/**
* RunInstance
* @param instance *Instance, ctx, tags et.Json, await bool, userId string
* @return et.Json, error
**/
func (s *WorkFlow) RunInstance(instance *Instance, ctx, tags et.Json, await bool, userId string) (et.Json, error) {
	defer func() {
		key := fmt.Sprintf("instance:%s", instance.ID)
		cache.Delete(key)
	}()

	instance.UserId = userId
	instance.setTag(tags)
	result, err := instance.Run(ctx, await)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Run
* @param tag, triggerTag, id, code string, ctx, tags et.Json, await bool, userId string
* @return *Instance, error
**/
func (s *WorkFlow) Run(tag, triggerTag, id, code string, ctx, tags et.Json, await bool, userId string) (et.Json, error) {
	instance, err := s.GetInstance(tag, triggerTag, id, code, tags, userId)
	if err != nil {
		return et.Json{}, err
	}

	return s.RunInstance(instance, ctx, tags, await, userId)
}
