package jwf

import (
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

const (
	SYSTEM   Status = "system"
	ACTIVE   Status = "active"
	ARCHIVED Status = "archived"
	// Instance Status
	CREATED  Status = "created"
	PENDING  Status = "pending"
	RUNNING  Status = "running"
	ROLLBACK Status = "rollback"
	DONE     Status = "done"
	FAILED   Status = "failed"
	CANCEL   Status = "cancel"
	STOP     Status = "stop"
)

var (
	ErrorInstanceNotFound                 = errors.New(MSG_INSTANCE_NOT_FOUND)
	FlowStatusList        map[Status]bool = map[Status]bool{
		CREATED:  true,
		PENDING:  true,
		RUNNING:  true,
		ROLLBACK: true,
		DONE:     true,
		FAILED:   true,
		CANCEL:   true,
		STOP:     true,
	}
)

type Result struct {
	StepId string  `json:"step_id"`
	Ctx    et.Json `json:"ctx"`
	Result et.Json `json:"result"`
	Error  string  `json:"error"`
}

type Current struct {
	Source     *Step `json:"source"`
	Target     *Step `json:"target"`
	Error      *Step `json:"error"`
	Index      int   `json:"index"`
	IsFinished bool  `json:"is_finished"`
}

type Instance struct {
	StartedAt  time.Time                                       `json:"started_at"`
	UpdatedAt  time.Time                                       `json:"updated_at"`
	DoneAt     time.Time                                       `json:"done_at"`
	TenantId   string                                          `json:"tenant_id"`
	ProjectId  string                                          `json:"project_id"`
	ID         string                                          `json:"id"`
	FlowId     string                                          `json:"flow_id"`
	Code       string                                          `json:"code"`
	Title      string                                          `json:"title"`
	Status     Status                                          `json:"status"`
	Ctx        et.Json                                         `json:"ctx"`
	Ctxs       map[string]et.Json                              `json:"ctxs"`
	Results    map[string]*Result                              `json:"results"`
	Params     et.Json                                         `json:"params"`
	Tags       et.Json                                         `json:"tags"`
	TriggerTag string                                          `json:"trigger_tag"`
	Trigger    *Trigger                                        `json:"trigger"`
	Current    *Current                                        `json:"current"`
	IsDone     bool                                            `json:"is_done"`
	IsStop     bool                                            `json:"is_stop"`
	Rollbacks  bool                                            `json:"rollbacks"`
	AuditLog   []et.Json                                       `json:"audit_log"`
	isDebug    bool                                            `json:"-"`
	isChanged  bool                                            `json:"-"`
	store      Store                                           `json:"-"`
	workflow   *WorkFlow                                       `json:"-"`
	flow       *Flow                                           `json:"-"`
	bindings   map[string]interface{}                          `json:"-"`
	resilience *resilience.Resilience                          `json:"-"`
	onSave     []func(instance *Instance, userId string) error `json:"-"`
	onDelete   []func(instance *Instance, userId string) error `json:"-"`
	mu         sync.Mutex                                      `json:"-"`
}

/**
* newInstance
* @param params InstanceParams
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(projectId, flowId, triggerTag, userId string) (*Instance, error) {
	flow, err := s.getFlow(flowId)
	if err != nil {
		return nil, err
	}

	trigger, exists := flow.getTrigger(triggerTag)
	if !exists {
		return nil, errors.New(MSG_TRIGGER_NOT_FOUND)
	}

	code := ""
	if s.store == nil {
		var err error
		code, err = s.store.GetCode(flow.Tag)
		if err != nil {
			return nil, err
		}
	}

	title := flow.Title
	if code != "" {
		title = fmt.Sprintf("%s %s", flow.Title, code)
	}

	now := timezone.Now()
	id := reg.ULID()
	result := &Instance{
		StartedAt:  now,
		TenantId:   s.TenantId,
		ProjectId:  projectId,
		ID:         id,
		FlowId:     flowId,
		Code:       code,
		Title:      title,
		Ctx:        et.Json{},
		Ctxs:       make(map[string]et.Json),
		Results:    make(map[string]*Result),
		Params:     et.Json{},
		Tags:       et.Json{},
		TriggerTag: triggerTag,
		Trigger:    trigger,
		IsDone:     false,
		IsStop:     false,
		AuditLog:   make([]et.Json, 0),
		store:      s.store,
		workflow:   s,
		flow:       flow,
		bindings:   make(map[string]interface{}),
		onSave:     make([]func(instance *Instance, userId string) error, 0),
		onDelete:   make([]func(instance *Instance, userId string) error, 0),
	}
	for k, v := range s.bindings {
		result.bindings[k] = v
	}
	result.
		OnSave(func(instance *Instance, userId string) error {
			key := fmt.Sprintf("instance:%s", instance.ID)
			event.Publish(key, instance.ToJson())
			return nil
		}).
		OnDelete(func(instance *Instance, userId string) error {
			key := fmt.Sprintf("instance:%s:delete", instance.ID)
			event.Publish(key, et.Json{
				"id": instance.ID,
			})
			return nil
		})
	result.addAuditLog(userId, "new_instance")
	result.setStatus(CREATED, userId)
	return result, nil
}

/**
* getInstance
* @param id, userId string
* @return *Instance, error
**/
func (s *WorkFlow) getInstance(id, userId string) (*Instance, error) {
	if s.store == nil {
		return nil, errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	var result *Instance
	exists, err := s.store.Get("instance", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrorInstanceNotFound
	}

	flow, err := s.getFlow(result.FlowId)
	if err != nil {
		return nil, err
	}

	trigger, exists := flow.getTrigger(result.TriggerTag)
	if !exists {
		return nil, errors.New(MSG_TRIGGER_NOT_FOUND)
	}

	result.store = s.store
	result.workflow = s
	result.flow = flow
	result.Trigger = trigger
	result.isDebug = s.isDebug
	result.onSave = make([]func(instance *Instance, userId string) error, 0)
	result.onDelete = make([]func(instance *Instance, userId string) error, 0)
	result.bindings = make(map[string]interface{})
	for k, v := range s.bindings {
		result.bindings[k] = v
	}
	result.
		OnSave(func(instance *Instance, userId string) error {
			key := fmt.Sprintf("instance:%s", instance.ID)
			event.Publish(key, instance.ToJson())
			return nil
		}).
		OnDelete(func(instance *Instance, userId string) error {
			key := fmt.Sprintf("instance:%s:delete", instance.ID)
			event.Publish(key, et.Json{
				"id": instance.ID,
			})
			return nil
		})
	result.addAuditLog(userId, "get_instance")
	return result, nil
}

/**
* deleteInstance
* @param id string, userId string
* @return error
**/
func (s *WorkFlow) deleteInstance(id, userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	instance, err := s.getInstance(id, userId)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("instance:%s:status", id)
	cache.Delete(key)

	err = s.store.Delete("instance", id)
	if err != nil {
		return err
	}

	for _, onDelete := range instance.onDelete {
		err := onDelete(instance, userId)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* addAuditLog
* @param userId string, action interface{}
**/
func (s *Instance) addAuditLog(userId string, action interface{}) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* OnSave
* @param fn func(instance *Instance, userId string) error
* @return *Jrex
**/
func (s *Instance) OnSave(fn func(instance *Instance, userId string) error) *Instance {
	if s.onSave == nil {
		s.onSave = make([]func(instance *Instance, userId string) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(instance *Instance, userId string) error
* @return *Step
**/
func (s *Instance) OnDelete(fn func(instance *Instance, userId string) error) *Instance {
	if s.onDelete == nil {
		s.onDelete = make([]func(instance *Instance, userId string) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Instance) save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("instance", s.ID, s.TenantId, s.ProjectId, s, userId)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s, userId)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	return et.Json{
		"started_at":  timezone.Format(s.StartedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"done_at":     timezone.Format(s.DoneAt, timezone.RFC3339),
		"tenant_id":   s.TenantId,
		"id":          s.ID,
		"code":        s.Code,
		"title":       s.Title,
		"status":      s.Status,
		"ctx":         s.Ctx,
		"ctxs":        s.Ctxs,
		"results":     s.Results,
		"params":      s.Params,
		"tags":        s.Tags,
		"trigger_tag": s.TriggerTag,
		"trigger":     s.Trigger,
		"current":     s.Current,
		"is_done":     s.IsDone,
		"is_stop":     s.IsStop,
		"audit_log":   s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *Instance) ToString() string {
	return s.ToJson().ToString()
}

/**
* pushStatus
* @return *Instance
**/
func (s *Instance) pushStatus(status Status) *Instance {
	key := fmt.Sprintf("instance:%s:status", s.ID)
	cache.SetObject(key, status, s.flow.TimeAwait)
	return s
}

/**
* getStatus
* @return Status
**/
func (s *Instance) getStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status, userId string) error {
	curStatus := s.getStatus()
	if curStatus == status {
		return nil
	}

	s.pushStatus(status)
	s.UpdatedAt = timezone.Now()
	s.mu.Lock()
	s.Status = status
	s.mu.Unlock()
	switch status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	}

	return s.save(userId)
}

/**
* setTrace
* @param step int, result et.Json, err error
* @return error
**/
func (s *Instance) setTrace(stepId string, result et.Json, err error, userId string) error {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	s.addAuditLog(userId, et.Json{
		"action":  "set_trace",
		"step_id": stepId,
		"ctx":     s.Ctx,
		"result":  result,
		"error":   errMessage,
	})
	return s.save(userId)
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error, userId string) (et.Json, error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	stepId := ""
	if s.Current != nil && s.Current.Source != nil {
		stepId = s.Current.Source.ID
	}

	if stepId == "" {
		return result, err
	}

	s.Results[stepId] = &Result{
		StepId: stepId,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}

	if err != nil {
		s.setStatus(FAILED, userId)
		step := ""
		if s.Current != nil && s.Current.Source != nil {
			step = s.Current.Source.Title
		}
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, step, err.Error())
	} else {
		s.pushStatus(s.Status)
	}

	return result, err
}

/**
* setTag
* @param tags et.Json
* @return et.Json
**/
func (s *Instance) setTag(tags et.Json) et.Json {
	maps.Copy(s.Tags, tags)
	return s.Tags
}

/**
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	maps.Copy(s.Ctx, ctx)
	stepId := ""
	if s.Current != nil && s.Current.Source != nil {
		stepId = s.Current.Source.ID
		s.Ctxs[stepId] = ctx
	}
	return s.Ctx
}

/**
* setParams
* @param params et.Json
* @return et.Json
**/
func (s *Instance) SetParams(params et.Json) *Instance {
	maps.Copy(s.Params, params)
	return s
}

/**
* next
* @return bool
**/
func (s *Instance) next() bool {
	if s.IsStop {
		return false
	}

	if s.IsDone {
		return false
	}

	status := s.getStatus()
	if status == CANCEL {
		return false
	} else if status == STOP {
		return false
	}

	if s.Current == nil {
		current, exists := s.flow.getCurrent(s.Trigger.StartId, s.CurrentIndex)
		if !exists {
			return false
		}
		s.Current = current
	} else {
		target := s.Current.Target
		if s.Rollbacks {
			target = s.Current.Error
		}

		if target == nil {
			return false
		}

		current, exists := s.flow.getCurrent(target.ID, s.CurrentIndex)
		if !exists {
			return false
		}
		s.Current = current
	}

	return true
}

/**
* run
* @param ctx et.Json, userId string
* @return et.Json, error
**/
func (s *Instance) run(ctx et.Json, userId string) (et.Json, error) {
	var err error
	defer func() {
		s.setTrace(s.Current.Source.ID, ctx, err, userId)
	}()

	status := s.getStatus()
	if status == DONE {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
		return et.Json{}, err
	} else if status == RUNNING {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
		return et.Json{}, err
	} else if status == ROLLBACK {
		err = fmt.Errorf(MSG_INSTANCE_ROLLBACK, s.ID)
		return et.Json{}, err
	} else if status == CANCEL {
		err = fmt.Errorf(MSG_INSTANCE_CANCEL, s.ID)
		return et.Json{}, err
	}

	var result et.Json
	for s.next() {
		step := s.Current.Source
		if step == nil {
			return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
		}

		ctx = s.setCtx(ctx)
		result, err = step.run(s, ctx, userId)
		if err != nil {
			result, err = s.runResilence(ctx, err, userId)
			if err != nil {
				result, err := s.rollback(ctx, err, userId)
				if err != nil {
					return result, err
				}
			}
		}

		s.setResult(result, err, userId)

		if s.IsDone {
			return result, nil
		}

		if s.IsStop || step.Stop {
			return result, nil
		}
	}

	return result, err
}

/**
* runResilence
* @return (bool, error)
**/
func (s *Instance) runResilence(ctx et.Json, err error, userId string) (et.Json, error) {
	if s.flow.TotalAttempts == 0 {
		return et.Json{}, err
	}

	if s.resilience == nil {
		resilience, err := resilience.New(s.workflow.resilienceStore)
		if err != nil {
			return et.Json{}, err
		}
		s.resilience = resilience
	}

	description := fmt.Sprintf("flow: %s,  %s", s.flow.Title, s.flow.Description)
	resilence := s.resilience.LoadInstance(resilience.Params{
		TenantId:      s.TenantId,
		Id:            s.ID,
		Tag:           "workflow",
		Description:   description,
		OwnerId:       s.ProjectId,
		TotalAttempts: s.flow.TotalAttempts,
		Interval:      s.flow.TimeAttempts,
		Tags:          s.Tags,
		UserId:        userId,
		Fn:            s.run,
		FnArgs:        []interface{}{ctx, userId},
	})
	res, err := resilence.Run(userId)
	if err != nil {
		return et.Json{}, err
	}

	if len(res) == 0 {
		return et.Json{}, errors.New(MSG_RESILIENCE_NO_RESULT)
	}

	result, ok := res[0].(et.Json)
	if !ok {
		return et.Json{}, errors.New(MSG_RESILIENCE_NO_RESULT)
	}

	return result, nil
}

/**
* rollback
* @return et.Json, error
**/
func (s *Instance) rollback(result et.Json, err error, userId string) (et.Json, error) {
	if s.Rollbacks {
		return result, err
	}
	s.Rollbacks = true
	s.setResult(result, err, userId)

	return result, err
}
