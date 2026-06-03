package workflow

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

const (
	PENDING  Status = "pending"
	RUNNING  Status = "running"
	ROLLBACK Status = "rollback"
	DONE     Status = "done"
	FAILED   Status = "failed"
	CANCEL   Status = "cancel"
)

var FlowStatusList map[Status]bool = map[Status]bool{
	PENDING:  true,
	RUNNING:  true,
	ROLLBACK: true,
	DONE:     true,
	FAILED:   true,
	CANCEL:   true,
}

type Instance struct {
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	ID          string               `json:"id"`
	Tag         string               `json:"tag"`
	OwnerId     string               `json:"owner_id"`
	CreatedBy   string               `json:"created_by"`
	UpdatedBy   string               `json:"updated_by"`
	Ctx         et.Json              `json:"ctx"`
	Ctxs        map[int]et.Json      `json:"ctxs"`
	Results     map[int]*Result      `json:"results"`
	Rollbacks   map[int]*Result      `json:"rollbacks"`
	Params      et.Json              `json:"params"`
	Traces      []et.Json            `json:"traces"`
	CheckList   []*CheckList         `json:"check_list"`
	Status      Status               `json:"status"`
	DoneAt      time.Time            `json:"done_at"`
	Tags        et.Json              `json:"tags"`
	Resilence   *resilience.Instance `json:"resilence"`
	Steper      *Steper              `json:"steeper"`
	CurrentStep int                  `json:"current_step"`
	IsDone      bool                 `json:"is_done"`
	IsStop      bool                 `json:"is_stop"`
	step        *Step                `json:"-"`
	goToStep    bool                 `json:"-"`
	flow        *Flow                `json:"-"`
	workflow    *WorkFlow            `json:"-"`
	isDebug     bool                 `json:"-"`
	mu          sync.Mutex           `json:"-"`
}

/**
* newInstance
* @param steper *Steper, id, ownerId, userName string
* @return *Instance
 */
func newInstance(steper *Steper, id, ownerId, userName string) *Instance {
	if id == "" {
		id = reg.GenUUId(steper.Tag)
	}
	if ownerId == "" {
		ownerId = id
	}
	now := timezone.Now()
	return &Instance{
		CreatedAt:   now,
		UpdatedAt:   now,
		ID:          id,
		Tag:         steper.flow.Tag,
		OwnerId:     ownerId,
		CreatedBy:   userName,
		UpdatedBy:   userName,
		Ctx:         et.Json{},
		Ctxs:        make(map[int]et.Json),
		Results:     make(map[int]*Result),
		Rollbacks:   make(map[int]*Result),
		Params:      et.Json{},
		Traces:      []et.Json{},
		CheckList:   []*CheckList{},
		Status:      PENDING,
		Tags:        et.Json{},
		Steper:      steper,
		CurrentStep: -1,
		IsDone:      false,
		IsStop:      false,
		goToStep:    false,
		flow:        steper.flow,
		workflow:    steper.flow.workflow,
		isDebug:     steper.flow.workflow.isDebug,
		mu:          sync.Mutex{},
	}
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.workflow != nil && s.workflow.store != nil {
		err := s.workflow.store.Set(s.ID, s.Tag, s.OwnerId, s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_INSTANCE_SET, data)

	return nil
}

/**
* delete
* @return error
**/
func (s *Instance) delete() error {
	if s.workflow != nil && s.workflow.store != nil {
		err := s.workflow.store.Delete(s.Tag)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_INSTANCE_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* up
* @param flow *Flow
**/
func (s *Instance) up(flow *Flow) {
	s.flow = flow
	s.workflow = flow.workflow
	s.isDebug = flow.workflow.isDebug
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	result := et.Json{
		"created_at":   s.CreatedAt,
		"updated_at":   s.UpdatedAt,
		"id":           s.ID,
		"tag":          s.Tag,
		"owner_id":     s.OwnerId,
		"created_by":   s.CreatedBy,
		"updated_by":   s.UpdatedBy,
		"ctx":          s.Ctx,
		"ctxs":         s.Ctxs,
		"results":      s.Results,
		"rollbacks":    s.Rollbacks,
		"params":       s.Params,
		"traces":       s.Traces,
		"check_list":   s.CheckList,
		"status":       s.Status,
		"tags":         s.Tags,
		"steper":       s.Steper,
		"current_step": s.CurrentStep,
		"is_done":      s.IsDone,
		"is_stop":      s.IsStop,
	}

	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result
}

/**
* ToString
* @return string
**/
func (s *Instance) ToString() string {
	return s.ToJson().ToString()
}

/**
* setTag
* @param key string, value interface{}
* @return et.Json
**/
func (s *Instance) SetTag(key string, value interface{}) et.Json {
	s.Tags[key] = value
	return s.Tags
}

/**
* putTag
* @param tags et.Json
* @return et.Json
**/
func (s *Instance) putTag(tags et.Json) et.Json {
	maps.Copy(s.Tags, tags)
	return s.Tags
}

/**
* setTrace
* @param step int, result et.Json, err error
* @return error
**/
func (s *Instance) setTrace(step int, result et.Json, err error) error {
	s.Traces = append(s.Traces, et.Json{
		"step":   step,
		"ctx":    s.Ctx,
		"result": result,
		"error":  err,
	})
	return s.save()
}

/**
* SetParam
* @param key string, value interface{}
**/
func (s *Instance) SetParam(key string, value interface{}) et.Json {
	s.Params[key] = value
	return s.Params
}

/**
* PutParam
* @param value et.Json
* @return et.Json
**/
func (s *Instance) PutParam(value et.Json) et.Json {
	maps.Copy(s.Params, value)
	return s.Params
}

/**
* SetCheckList
* @param tag string, ok bool, data et.Json
* @return error
**/
func (s *Instance) SetCheckList(tag string, ok bool, data et.Json) error {
	idx := slices.IndexFunc(s.flow.CheckList, func(check *CheckList) bool { return check.Tag == tag })
	if idx != -1 {
		check := s.flow.CheckList[idx]
		s.CheckList = append(s.CheckList, &CheckList{
			Tag:         check.Tag,
			Description: check.Description,
			Ok:          ok,
			Data:        data,
		})
		return nil
	}
	return fmt.Errorf(MSG_CHECK_LIST_NOT_FOUND)
}

/**
* setStatus
* @param status FlowStatus
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	if s.Status == status {
		return nil
	}

	s.Status = status
	s.UpdatedAt = timezone.Now()
	switch s.Status {
	case DONE:
		s.DoneAt = s.UpdatedAt
		s.IsDone = true
	default:
		logs.Logf(packageName, MSG_INSTANCE_STATUS, s.ID, s.Tag, s.Status, s.CurrentStep)
	}

	return s.save()
}

/**
* setResult
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setResult(result et.Json, err error) (et.Json, error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}

	attempt := 0
	if s.Resilence != nil {
		attempt = s.Resilence.Attempt
	}
	res := &Result{
		Step:    s.CurrentStep,
		Ctx:     s.Ctx,
		Attempt: attempt,
		Result:  result,
		Error:   errMessage,
	}
	s.Results[s.CurrentStep] = res
	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ERROR, s.ID, s.Tag, s.CurrentStep, err.Error())
	}

	return result, err
}

/**
* setRollback
* @param result et.Json, err error
* @return et.Json, error
**/
func (s *Instance) setRollback(result et.Json, err error) (et.Json, error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}

	attempt := 0
	if s.Resilence != nil {
		attempt = s.Resilence.Attempt
	}
	res := &Result{
		Step:    s.CurrentStep,
		Ctx:     s.Ctx,
		Attempt: attempt,
		Result:  result,
		Error:   errMessage,
	}
	s.Rollbacks[s.CurrentStep] = res
	if err != nil {
		s.setStatus(FAILED)
		logs.Logf(packageName, MSG_INSTANCE_ROLLBACK_ERROR, s.ID, s.Tag, s.CurrentStep, err.Error())
	}

	return result, err
}

/**
* SetCtx
* @param ctx et.Json, step int
* @return et.Json
**/
func (s *Instance) SetCtx(ctx et.Json, step int) et.Json {
	maps.Copy(s.Ctx, ctx)
	s.Ctxs[step] = s.Ctx.Clone()
	return s.Ctx
}

/**
* getCtx
* @param idx int
* @return et.Json
**/
func (s *Instance) GetCtx(idx int) et.Json {
	result, ok := s.Ctxs[idx]
	if !ok {
		return et.Json{}
	}

	return result
}

/**
* setStop
* @param result et.Json
* @return (et.Json, error)
**/
func (s *Instance) setStop(result et.Json) (et.Json, error) {
	err := s.setStatus(PENDING)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* SetCurrentStep
* @param step int
**/
func (s *Instance) SetCurrentStep(step int) {
	if step < 0 {
		step = 0
	}
	s.CurrentStep = step - 1
}

/**
* Next
* @return bool
**/
func (s *Instance) Next() bool {
	if s.isStop() {
		return false
	}

	s.CurrentStep++
	step, ok := s.Steper.GetStep(s.CurrentStep)
	if !ok {
		return false
	}

	s.step = step
	return s.step != nil
}

/**
* run
* @param ctx et.Json
* @return et.Json, error
**/
func (s *Instance) run(ctx et.Json) (et.Json, error) {
	var err error
	defer func() {
		s.setTrace(s.CurrentStep, ctx, err)
	}()

	if s.Status == DONE {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
		return et.Json{}, err
	} else if s.Status == RUNNING {
		err = fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
		return et.Json{}, err
	} else if s.Status == ROLLBACK {
		err = fmt.Errorf(MSG_INSTANCE_ROLLBACK, s.ID)
		return et.Json{}, err
	} else if s.Status == CANCEL {
		err = fmt.Errorf(MSG_INSTANCE_CANCEL, s.ID)
		return et.Json{}, err
	}

	var result et.Json
	for s.Next() {
		ctx = s.SetCtx(ctx, s.CurrentStep)
		result, err = s.step.Run(s, ctx)
		if err != nil {
			result, err := s.rollback()
			if err != nil {
				return result, err
			}
		}

		if s.IsDone {
			return result, nil
		}

		if s.goToStep {
			s.goToStep = false
			continue
		}

		if s.isStop() || s.step.Stop {
			return s.setStop(result)
		}
	}

	return result, err
}

/**
* rollback
* @param step *Step, ctx et.Json
* @return et.Json, error
**/
func (s *Instance) rollback() (et.Json, error) {
	if s.flow.TotalAttempts > 0 {
		result, err := s.startResilence()
		if err == nil {
			return result, nil
		}
	}

	if s.Status == DONE {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_ALREADY_DONE, s.ID)
	} else if s.Status == RUNNING {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_ALREADY_RUNNING, s.ID)
	} else if s.Status == PENDING {
		return et.Json{}, fmt.Errorf(MSG_INSTANCE_PENDING, s.ID)
	}

	var err error
	var result et.Json
	for i := s.CurrentStep - 1; i >= 0; i-- {
		step, exists := s.Steper.GetStep(i)
		if !exists {
			continue
		}

		logs.Logf(packageName, MSG_INSTANCE_ROLLBACK_STEP, i)
		ctx := s.GetCtx(i)
		result, err = step.RunRollback(s, ctx)
		if err != nil {
			return et.Json{}, err
		}
	}

	return result, nil
}

/**
* startResilence
* @return (bool, error)
**/
func (s *Instance) startResilence() (et.Json, error) {
	description := fmt.Sprintf("flow: %s,  %s", s.flow.Name, s.flow.Description)
	s.Resilence = s.workflow.resilience.LoadInstance(s.ID, s.Tag, description, s.OwnerId, s.flow.TotalAttempts, s.flow.TimeAttempts, s.Tags, s.flow.Team, s.flow.Level, s.run, s.Ctx)
	if s.Resilence.Error != "" {
		return et.Json{}, errors.New(s.Resilence.Error)
	}
	s.Resilence.Run()
	result, ok := s.Resilence.Result.(et.Json)
	if !ok {
		return et.Json{}, nil
	}

	return result, nil
}

/**
* Done
* @return error
**/
func (s *Instance) Done() error {
	return s.setStatus(DONE)
}

/**
* Stop
* @param stop bool
* @return error
**/
func (s *Instance) Stop(stop bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsStop = stop
	return nil
}

/**
* isStop
* @return bool
**/
func (s *Instance) isStop() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.IsStop
}

/**
* GoToStep
* @param step int, message string, ctx et.Json
* @return error
**/
func (s *Instance) GoToStep(step int, message string, ctx et.Json) error {
	if step < 0 || step >= len(s.Steper.Steps) {
		return nil
	}

	s.goToStep = true
	s.SetCurrentStep(step)
	s.SetCtx(ctx, step)
	err := s.setStatus(RUNNING)
	if err != nil {
		return err
	}

	logs.Logf(packageName, MSG_INSTANCE_GOTO, s.ID, s.Tag, step, message)
	return nil
}

/**
* SetStatus
* @param status Status
* @return error
**/
func (s *Instance) SetStatus(status Status) error {
	return s.setStatus(status)
}
