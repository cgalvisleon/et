package jwf

import (
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
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
	SourceId   string `json:"source_id"`
	TargetId   string `json:"target_id"`
	ErrorId    string `json:"error_id"`
	Index      int    `json:"index"`
	IsFinished bool   `json:"is_finished"`
}

type Instance struct {
	StartedAt    time.Time                        `json:"started_at"`
	UpdatedAt    time.Time                        `json:"updated_at"`
	DoneAt       time.Time                        `json:"done_at"`
	WorkflowId   string                           `json:"workflow_id"`
	ProjectId    string                           `json:"project_id"`
	ID           string                           `json:"id"`
	FlowId       string                           `json:"flow_id"`
	Code         string                           `json:"code"`
	Title        string                           `json:"title"`
	Status       Status                           `json:"status"`
	Ctx          et.Json                          `json:"ctx"`
	Ctxs         map[string]et.Json               `json:"ctxs"`
	Results      map[string]*Result               `json:"results"`
	Tags         et.Json                          `json:"tags"`
	TriggerTag   string                           `json:"trigger_tag"`
	Trigger      *Trigger                         `json:"trigger"`
	Current      *Step                            `json:"current"`
	CurrentIndex int                              `json:"current_index"`
	IsDone       bool                             `json:"is_done"`
	IsStop       bool                             `json:"is_stop"`
	AuditLog     []et.Json                        `json:"audit_log"`
	isDebug      bool                             `json:"-"`
	isChanged    bool                             `json:"-"`
	store        Store                            `json:"-"`
	workflow     *WorkFlow                        `json:"-"`
	flow         *Flow                            `json:"-"`
	bindings     map[string]interface{}           `json:"-"`
	resilience   *resilience.Resilience           `json:"-"`
	onSave       []func(instance *Instance) error `json:"-"`
	onDelete     []func(instance *Instance) error `json:"-"`
	mu           sync.Mutex                       `json:"-"`
}

/**
* newInstance
* @param params InstanceParams
* @return *Instance, error
**/
func (s *WorkFlow) newInstance(projectId, flowId, triggerTag, userId string) (*Instance, error) {
	flow, err := s.loadFlow(flowId)
	if err != nil {
		return nil, err
	}

	trigger, exists := flow.getTrigger(triggerTag)
	if !exists {
		return nil, errors.New(MSG_TRIGGER_NOT_FOUND)
	}

	code := ""
	if s.store != nil {
		var err error
		code, err = s.store.GenSerie(flow.Tag)
		if err != nil {
			return nil, err
		}
	}

	title := flow.Title
	if code != "" {
		title = fmt.Sprintf("%s %s", flow.Title, code)
	}

	now := timezone.Now()
	id := reg.UUID()
	result := &Instance{
		StartedAt:  now,
		WorkflowId: s.ID,
		ProjectId:  projectId,
		ID:         id,
		FlowId:     flowId,
		Code:       code,
		Title:      title,
		Ctx:        et.Json{},
		Ctxs:       make(map[string]et.Json),
		Results:    make(map[string]*Result),
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
		onSave:     make([]func(instance *Instance) error, 0),
		onDelete:   make([]func(instance *Instance) error, 0),
		mu:         sync.Mutex{},
	}
	result.addAuditLog(userId, "new_instance")
	result.up(flow)
	result.setStatus(CREATED)
	return result, nil
}

/**
* getInstance
* @param id, userId string
* @return *Instance, error
**/
func (s *WorkFlow) getInstance(id, userId string) (*Instance, error) {
	if id != "" {
		key := fmt.Sprintf("instance:%s:status", id)
		status, err := cache.Get(key, "")
		if err != nil {
			return nil, err
		}
		if status != "" {
			return nil, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, status)
		}
	}

	if s.store == nil {
		return nil, ErrorInstanceNotFound
	}

	var result *Instance
	exists, err := s.store.Get("instance", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrorInstanceNotFound
	}

	flow, err := s.loadFlow(result.FlowId)
	if err != nil {
		return nil, err
	}

	trigger, exists := flow.getTrigger(result.TriggerTag)
	if !exists {
		return nil, errors.New(MSG_TRIGGER_NOT_FOUND)
	}
	result.Trigger = trigger

	result.addAuditLog(userId, "get_instance")
	return result.up(flow), nil
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
		err := onDelete(instance)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* up
* @return *Instance
**/
func (s *Instance) up(flow *Flow) *Instance {
	s.flow = flow
	s.store = flow.store
	s.workflow = flow.workflow
	s.isDebug = flow.isDebug
	s.onSave = make([]func(instance *Instance) error, 0)
	s.onDelete = make([]func(instance *Instance) error, 0)
	for k, v := range flow.workflow.bindings {
		s.bindings[k] = v
	}
	s.bindings["goTo"] = func(idx int) {
		s.setCurrentIndex(idx)
	}
	s.OnSave(func(instance *Instance) error {
		instance.pushInstance()
		return nil
	})
	s.OnDelete(func(instance *Instance) error {
		key := fmt.Sprintf("instance:%s:delete", instance.ID)
		event.Publish(key, et.Json{
			"id": instance.ID,
		})
		return nil
	})
	return s
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
* OnSave
* @param fn func(instance *Instance) error
* @return *Jrex
**/
func (s *Instance) OnSave(fn func(instance *Instance) error) *Instance {
	if s.onSave == nil {
		s.onSave = make([]func(instance *Instance) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(instance *Instance) error
* @return *Step
**/
func (s *Instance) OnDelete(fn func(instance *Instance) error) *Instance {
	if s.onDelete == nil {
		s.onDelete = make([]func(instance *Instance) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	if s.store == nil {
		return errors.New(MSG_WORKFLOW_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("instances", s.ID, s.WorkflowId, s)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s)
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
		"workflow_id": s.WorkflowId,
		"project_id":  s.ProjectId,
		"id":          s.ID,
		"code":        s.Code,
		"title":       s.Title,
		"status":      s.Status,
		"ctx":         s.Ctx,
		"ctxs":        s.Ctxs,
		"results":     s.Results,
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
* pushInstance
* @return error
**/
func (s *Instance) pushInstance() {
	key := fmt.Sprintf("instance:%s", s.ID)
	event.Publish(key, s.ToJson())
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
func (s *Instance) setStatus(status Status) error {
	curStatus := s.getStatus()
	if curStatus == status {
		return nil
	}

	s.pushStatus(status)
	s.pushInstance()
	s.UpdatedAt = timezone.Now()
	s.mu.Lock()
	s.Status = status
	s.IsStop = status == STOP
	s.mu.Unlock()
	switch status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	}

	return s.save()
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
	return s.save()
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) *Instance {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	stepId := ""
	if s.Current != nil {
		stepId = s.Current.ID
	}

	if stepId == "" {
		return s
	}

	s.Results[stepId] = &Result{
		StepId: stepId,
		Ctx:    s.Ctx,
		Result: result,
		Error:  errMessage,
	}

	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.FlowId, stepId, err.Error())
	} else {
		s.pushStatus(s.Status)
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.FlowId, stepId, s.Status)
	}

	s.pushInstance()
	return s
}

/**
* setTag
* @param tags et.Json
* @return et.Json
**/
func (s *Instance) setTag(tags et.Json) et.Json {
	maps.Copy(s.Tags, tags)
	s.pushInstance()
	return s.Tags
}

/**
* setCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) setCtx(ctx et.Json) et.Json {
	maps.Copy(s.Ctx, ctx)
	if s.Current != nil {
		stepId := s.Current.ID
		s.Ctxs[stepId] = ctx
	}
	s.pushInstance()
	return s.Ctx
}

/**
* setCurrent
* @param step *Step
* @return error
**/
func (s *Instance) setCurrent(step *Step) {
	s.Current = step
	s.pushInstance()
}

/**
* setCurrentIndex
* @param idx int
* @return *Instance
**/
func (s *Instance) setCurrentIndex(idx int) {
	s.CurrentIndex = idx
	s.pushInstance()
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
		step, exists := s.flow.getStep(s.Trigger.StartId)
		if !exists {
			return false
		}
		s.setCurrent(step)
	} else {
		target, exists := s.flow.getTarget(s.Current.ID, s.CurrentIndex)
		if !exists {
			return false
		}
		s.setCurrent(target)
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
		s.setTrace(s.Current.ID, ctx, err, userId)
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
		step := s.Current
		if step == nil {
			return et.Json{}, errors.New(MSG_STEP_NOT_FOUND)
		}

		ctx = s.setCtx(ctx)
		result, err = step.run(s, ctx)
		if err != nil {
			result, err = s.runResilence(ctx, err, userId)
			if err != nil {
				result, err = s.runError(ctx, err)
			}
		}

		s.setResult(result, err)

		if err != nil {
			return result, err
		}

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
		resilience, err := resilience.New(s.workflow.store)
		if err != nil {
			return et.Json{}, err
		}
		s.resilience = resilience
	}

	description := fmt.Sprintf("flow: %s,  %s", s.flow.Title, s.flow.Description)
	resilence := s.resilience.LoadInstance(resilience.Params{
		Id:            s.ID,
		Tag:           "workflow",
		Description:   description,
		TotalAttempts: s.flow.TotalAttempts,
		Interval:      s.flow.TimeAttempts,
		Tags:          s.Tags,
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
func (s *Instance) runError(ctx et.Json, err error) (et.Json, error) {
	step, exists := s.flow.getError(s.Current.ID, 0)
	if !exists {
		return et.Json{}, err
	}

	result, errStep := step.run(s, ctx)
	if errStep != nil {
		return et.Json{}, errStep
	}

	return result, err
}
