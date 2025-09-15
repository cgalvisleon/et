package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

var (
	errorInstanceNotFound = errors.New(MSG_INSTANCE_NOT_FOUND)
)

const packageName = "workflow"

type instanceFn func(instanceId, tag string, startId int, tags, ctx et.Json, createdBy string) (et.Json, error)

type Awaiting struct {
	CreatedAt  time.Time     `json:"created_at"`
	ExecutedAt time.Time     `json:"executed_at"`
	Id         string        `json:"id"`
	fn         interface{}   `json:"-"`
	fnArgs     []interface{} `json:"-"`
}

func (s *Awaiting) ToJson() et.Json {
	return et.Json{
		"created_at":  s.CreatedAt,
		"id":          s.Id,
		"args":        s.fnArgs,
		"executed_at": s.ExecutedAt,
	}
}

type WorkFlows struct {
	Flows         map[string]*Flow     `json:"flows"`
	Instances     map[string]*Instance `json:"instances"`
	LimitRequests int                  `json:"limit_requests"`
	AwaitingList  []*Awaiting          `json:"awaiting_list"`
	Results       map[string]et.Json   `json:"results"`
	RetentionTime time.Duration        `json:"retention_time"`
	count         int                  `json:"-"`
	mu            sync.Mutex           `json:"-"`
}

/**
* newWorkFlows
* @return *WorkFlows
**/
func newWorkFlows() *WorkFlows {
	retentionTime := envar.GetInt("WORKFLOW_RETENTION_TIME", 10)
	result := &WorkFlows{
		Flows:         make(map[string]*Flow),
		Instances:     make(map[string]*Instance),
		LimitRequests: envar.GetInt("WORKFLOW_LIMIT_REQUESTS", 0),
		AwaitingList:  make([]*Awaiting, 0),
		Results:       make(map[string]et.Json),
		RetentionTime: time.Duration(retentionTime) * time.Minute,
		count:         0,
		mu:            sync.Mutex{},
	}

	return result
}

/**
* healthCheck
* @return bool
**/
func (s *WorkFlows) healthCheck() bool {
	ok := resilience.HealthCheck()
	if !ok {
		return false
	}

	return true
}

/**
* instanceInc
**/
func (s *WorkFlows) instanceInc() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	logs.Logf(packageName, MSG_INSTANCE_INSTANCE_INC, s.count, s.LimitRequests)
}

/**
* instanceDec
**/
func (s *WorkFlows) instanceDec() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count--
	logs.Logf(packageName, MSG_INSTANCE_INSTANCE_DEC, s.count, s.LimitRequests)
}

/**
* instanceCount
* @return int
**/
func (s *WorkFlows) instanceCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.count
}

/**
* newInstance
* @param tag, id string, tags et.Json, startId int, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) newInstance(tag, id string, tags et.Json, startId int, createdBy string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	flow := s.Flows[tag]
	if flow == nil {
		return nil, fmt.Errorf(MSG_FLOW_NOT_FOUND)
	}

	now := timezone.NowTime()
	result := &Instance{
		Flow:       flow,
		workFlows:  s,
		CreatedAt:  now,
		UpdatedAt:  now,
		Id:         id,
		CreatedBy:  createdBy,
		Current:    startId,
		Ctx:        et.Json{},
		Ctxs:       make(map[int]et.Json),
		Results:    make(map[int]*Result),
		Rollbacks:  make(map[int]*Result),
		Tags:       tags,
		goTo:       -1,
		WorkerHost: workerHost,
	}
	result.setStatus(FlowStatusPending)
	s.Instances[id] = result

	return result, nil
}

/**
* loadInstance
* @param id string
* @return *Flow, error
**/
func (s *WorkFlows) loadInstance(id string) (*Instance, error) {
	if id == "" {
		return nil, fmt.Errorf(MSG_INSTANCE_ID_REQUIRED)
	}

	if s.Instances[id] != nil {
		return s.Instances[id], nil
	}

	if !cache.Exists(id) {
		return nil, errorInstanceNotFound
	}

	result := &Instance{}
	bt, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	src, err := cache.Get(id, string(bt))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(src), &result)
	if err != nil {
		return nil, err
	}

	flow := s.Flows[result.Tag]
	if flow == nil {
		return nil, fmt.Errorf(MSG_FLOW_NOT_FOUND)
	}

	result.Flow = flow
	result.setStatus(result.Status)
	s.Instances[id] = result

	return result, nil
}

/**
* runNextInBackground
**/
func (s *WorkFlows) runNextInBackground() {
	if len(s.AwaitingList) == 0 {
		return
	}

	req := s.AwaitingList[0]
	s.AwaitingList = s.AwaitingList[1:]
	logs.Logf(packageName, MSG_INSTANCE_RUN, req.Id, req.ToJson().ToString())
	req.ExecutedAt = timezone.NowTime()

	argsValues := make([]reflect.Value, len(req.fnArgs))
	for i, arg := range req.fnArgs {
		argsValues[i] = reflect.ValueOf(arg)
	}

	fn := reflect.ValueOf(req.fn)
	fnResult := fn.Call(argsValues)
	res := &resultFn{
		Result: fnResult[0].Interface().(et.Json),
		Error:  fnResult[1].Interface().(error),
	}

	key := fmt.Sprintf("workflow:result:%s", req.Id)
	src, err := res.Serialize()
	if err != nil {
		logs.Logf(packageName, "WorkFlows.done, Error serializing result:%s", err.Error())
	}
	cache.Set(key, src, s.RetentionTime)
	event.Publish(EVENT_WORKFLOW_RESULTS, res.ToJson())
}

/**
* getOrCreateInstance
* @param id, tag string, startId int, tags et.Json, createdBy string
* @return *Instance, error
**/
func (s *WorkFlows) getOrCreateInstance(id, tag string, startId int, tags et.Json, createdBy string) (*Instance, error) {
	id = reg.GetUUID(id)
	if result, err := s.loadInstance(id); err == nil {
		return result, nil
	} else if errors.Is(err, errorInstanceNotFound) {
		return s.newInstance(tag, id, tags, startId, createdBy)
	}

	return nil, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
}

/**
* instanceRun
* @param instanceId, tag string, startId int, tags, ctx et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlows) instanceRun(instanceId, tag string, startId int, tags, ctx et.Json, createdBy string) (et.Json, error) {
	s.instanceInc()
	if instanceId != "" {
		key := fmt.Sprintf("workflow:result:%s", instanceId)
		if cache.Exists(key) {
			scr, err := cache.Get(key, "")
			if err != nil {
				return et.Json{}, err
			}

			result, err := loadResultFn(scr)
			if err != nil {
				return et.Json{}, err
			}

			return result.Result, result.Error
		}
	}

	instance, err := s.getOrCreateInstance(instanceId, tag, startId, tags, createdBy)
	if err != nil {
		return et.Json{}, err
	}

	result, err := instance.run(ctx)
	if err != nil {
		return et.Json{}, err
	}

	if instance.isDebug {
		logs.Debugf("Flow instance:%s", instance.ToJson().ToString())
	}

	return result, err
}

/**
* instanceGoOn
* @param instanceId string, tags et.Json, ctx et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlows) instanceGoOn(instanceId string, tags et.Json, ctx et.Json, createdBy string) (et.Json, error) {
	s.instanceInc()
	if instanceId != "" {
		key := fmt.Sprintf("workflow:result:%s", instanceId)
		if cache.Exists(key) {
			scr, err := cache.Get(key, "")
			if err != nil {
				return et.Json{}, err
			}

			result, err := loadResultFn(scr)
			if err != nil {
				return et.Json{}, err
			}

			return result.Result, result.Error
		}
	}

	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return et.Json{}, err
	}

	instance.setTags(tags)
	instance.setCtx(ctx)
	instance.CreatedBy = createdBy
	result, err := instance.run(ctx)
	if err != nil {
		return et.Json{}, err
	}

	if instance.isDebug {
		logs.Debugf("Flow instance:%s", instance.ToJson().ToString())
	}

	return result, err
}

/**
* newFlow
* @param tag, version, name, description string, fn FnContext, stop bool, createdBy string
* @return *Flow
**/
func (s *WorkFlows) newFlow(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	flow := newFlow(tag, version, name, description, fn, stop, createdBy)
	s.Flows[tag] = flow

	return flow
}

/**
* newAwaiting
* @param instanceId string, fnArgs ...interface{}
* @return *Awaiting
**/
func (s *WorkFlows) newAwaiting(instanceId string, fnArgs ...interface{}) (et.Json, error) {
	instanceId = reg.GetUUID(instanceId)
	awaiting := &Awaiting{
		CreatedAt: time.Now(),
		Id:        instanceId,
		fn:        s.run,
		fnArgs:    fnArgs,
	}
	s.AwaitingList = append(s.AwaitingList, awaiting)
	event.Publish(EVENT_WORKFLOW_AWAITING, awaiting.ToJson())

	return et.Json{}, fmt.Errorf(MSG_WORKFLOW_LIMIT_REQUESTS, instanceId)
}

/**
* run
* @param instanceId, tag string, startId int, tags, ctx et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlows) run(instanceId, tag string, startId int, tags, ctx et.Json, createdBy string) (et.Json, error) {
	response := func(result et.Json, err error) (et.Json, error) {
		s.instanceDec()
		delete(s.Instances, instanceId)
		logs.Logf(packageName, MSG_WORKFLOW_DONE_INSTANCE, instanceId)
		go s.runNextInBackground()

		return result, err
	}

	if s.LimitRequests == 0 {
		return response(s.instanceRun(instanceId, tag, startId, tags, ctx, createdBy))
	}

	totalInstances := s.instanceCount()
	if totalInstances < s.LimitRequests {
		return response(s.instanceRun(instanceId, tag, startId, tags, ctx, createdBy))
	}

	return s.newAwaiting(instanceId, []interface{}{instanceId, tag, startId, tags, ctx, createdBy})
}

/**
* GoOn
* @param instanceId string, tags et.Json, ctx et.Json, createdBy string
* @return et.Json, error
**/
func (s *WorkFlows) goOn(instanceId string, tags et.Json, ctx et.Json, createdBy string) (et.Json, error) {
	response := func(result et.Json, err error) (et.Json, error) {
		s.instanceDec()
		delete(s.Instances, instanceId)
		logs.Logf(packageName, MSG_WORKFLOW_DONE_INSTANCE, instanceId)
		go s.runNextInBackground()

		return result, err
	}

	if s.LimitRequests == 0 {
		return response(s.instanceGoOn(instanceId, tags, ctx, createdBy))
	}

	totalInstances := s.instanceCount()
	if totalInstances < s.LimitRequests {
		return response(s.instanceGoOn(instanceId, tags, ctx, createdBy))
	}

	return s.newAwaiting(instanceId, []interface{}{instanceId, tags, ctx, createdBy})
}

/**
* Rollback
* @param instanceId string
* @return et.Json, error
**/

func (s *WorkFlows) rollback(instanceId string) (et.Json, error) {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return et.Json{}, err
	}

	result, err := instance.rollback(et.Json{}, nil)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* stop
* @param instanceId string
* @return error
**/
func (s *WorkFlows) stop(instanceId string) error {
	instance, err := s.loadInstance(instanceId)
	if err != nil {
		return err
	}

	return instance.Stop()
}

/**
* deleteFlow
* @param tag string
* @return bool
**/
func (s *WorkFlows) deleteFlow(tag string) bool {
	if s.Flows[tag] == nil {
		return false
	}

	flow := s.Flows[tag]
	event.Publish(EVENT_WORKFLOW_DELETE, flow.ToJson())
	delete(s.Flows, tag)

	return true
}
