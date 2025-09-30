package crontab

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	"github.com/robfig/cron/v3"
)

const (
	packageName    = "crontab"
	StatusAdded    = "added"
	StatusPending  = "pending"
	StatusStarting = "starting"
	StatusDone     = "done"
	StatusNotified = "notified"
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusExecuted = "executed"
	StatusRemoved  = "removed"
	StatusFailed   = "failed"
)

var (
	ErrJobExists = fmt.Errorf("job already exists")
)

type Job struct {
	Id          string         `json:"id"`
	Name        string         `json:"name"`
	Channel     string         `json:"channel"`
	Params      et.Json        `json:"params"`
	Spec        string         `json:"spec"`
	Started     bool           `json:"started"`
	Status      string         `json:"status"`
	Idx         int            `json:"idx"`
	NodeId      int64          `json:"node_id"`
	Attempts    int            `json:"attempts"`
	Repetitions int            `json:"repetitions"`
	fn          func(job *Job) `json:"-"`
	jobs        *Jobs          `json:"-"`
	isEventUp   bool           `json:"-"`
	mu          *sync.Mutex    `json:"-"`
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Job) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Job) ToJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* eventUp
* @return void
**/
func (s *Job) eventUp() {
	if s.isEventUp {
		return
	}

	key := fmt.Sprintf("crontab:%s", s.Id)
	err := event.Stack(key, func(msg event.Message) {
		data := msg.Data
		action := data.Str("action")
		switch action {
		case "start":
			s.Start()
		case "stop":
			s.Stop()
		case "remove":
			s.Remove()
		}
	})
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Job %s event up error:%s", s.Name, err.Error()))
	}

	s.isEventUp = true
	logs.Logf(packageName, fmt.Sprintf("Job %s event up", s.Name))
}

/**
* eventDown
* @return void
**/
func (s *Job) eventDown() {
	s.isEventUp = false
	key := fmt.Sprintf("crontab:%s", s.Id)
	event.Unsubscribe(key)
}

/**
* setStatus
* @param status string
* @return void
**/
func (s *Job) setStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	switch status {
	case StatusAdded:
		s.eventUp()
	case StatusRemoved:
		s.eventDown()
	case StatusNotified:
		s.Attempts++
	case StatusDone:
		if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
			s.Remove()
		}
	case StatusFailed:
		if s.jobs.Team == "" {
			break
		}

		data := s.ToJson()
		data.Set("team", s.jobs.Team)
		data.Set("level", s.jobs.Level)
		data.Set("message", fmt.Sprintf("CrontabJob %s failed id:%s node:%d attempt:%d", s.Name, s.Id, s.NodeId, s.Attempts))
		event.Publish(EVENT_CRONTAB_FAILED, data)
	}

	logs.Logf(packageName, fmt.Sprintf("Job %s status:%s id:%s node:%d attempt:%d", s.Name, s.Status, s.Id, s.NodeId, s.Attempts))
	event.Publish(EVENT_CRONTAB_STATUS, s.ToJson())
}

/**
* Start
* @return error
**/
func (s *Job) Start() error {
	if s.Started {
		return nil
	}

	if s.fn == nil {
		s.fn = func(job *Job) {
			err := event.Publish(job.Channel, job.ToJson())
			if err != nil {
				s.setStatus(StatusFailed)
			}
		}
	}
	fn := func() {
		if s.fn != nil && s.Started {
			s.setStatus(StatusNotified)
			s.fn(s)
		}
	}

	id, err := s.jobs.crontab.AddFunc(s.Spec, fn)
	if err != nil {
		return err
	}

	s.Idx = int(id)
	s.Started = true
	s.setStatus(StatusStarting)

	return nil
}

/**
* Stop
* @return error
**/
func (s *Job) Stop() {
	if !s.Started {
		return
	}

	s.Started = false
	s.setStatus(StatusStopped)

	time.AfterFunc(time.Second*1, func() {
		s.jobs.crontab.Remove(cron.EntryID(s.Idx))
		s.Idx = -1
	})
}

/**
* Remove
* @return error
**/
func (s *Job) Remove() error {
	return s.jobs.removeJob(s)
}

type Jobs struct {
	Id         string     `json:"id"`
	Team       string     `json:"team"`
	Level      string     `json:"level"`
	nodeId     int64      `json:"-"`
	jobs       []*Job     `json:"-"`
	crontab    *cron.Cron `json:"-"`
	storageKey string     `json:"-"`
	running    bool       `json:"-"`
	isServer   bool       `json:"-"`
}

func New() *Jobs {
	version := "v0.0.1"
	return &Jobs{
		Id:         utility.UUID(),
		Team:       envar.GetStr("Operation", "IRT_TEAM"),
		Level:      envar.GetStr("", "IRT_LEVEL"),
		jobs:       make([]*Job, 0),
		crontab:    cron.New(cron.WithSeconds()),
		storageKey: fmt.Sprintf("crontab_%s", version),
	}
}

/**
* load
* @return error
**/
func (s *Jobs) load() error {
	err := cache.Load()
	if err != nil {
		return err
	}

	err = event.Load()
	if err != nil {
		return err
	}

	s.nodeId = cache.Incr("crontab:nodes", time.Second*60)
	if s.nodeId == 1 {
		event.Stack(EVENT_CRONTAB_SERVER, func(msg event.Message) {
			logs.Logf(packageName, `Crontab server loaded`)
		})

		logs.Logf(packageName, `Crontab server loaded`)
	} else {
		event.Publish(EVENT_CRONTAB_SERVER, et.Json{
			"event": "event:crontab:startNode",
			"id":    s.Id,
		})
		logs.Logf(packageName, `Crontab  loaded`)
	}

	return nil
}

/**
* addJob
* @param id, name, spec, channel string, params et.Json, repetitions int, fn func(job *Job)
* @return *Job, error
**/
func (s *Jobs) addJob(id, name, spec, channel string, params et.Json, repetitions int, fn func(job *Job)) (*Job, error) {
	if !utility.ValidStr(name, 0, []string{"", " "}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	idx := s.indexJobByName(name)
	if idx != -1 {
		result := s.jobs[idx]
		return result, nil
	}

	id = reg.GetUUID(id)
	result := &Job{
		Id:          id,
		Name:        name,
		Channel:     channel,
		Params:      params,
		Spec:        spec,
		Started:     false,
		Idx:         len(s.jobs),
		NodeId:      s.nodeId,
		Repetitions: repetitions,
		fn:          fn,
		jobs:        s,
		mu:          &sync.Mutex{},
	}
	s.jobs = append(s.jobs, result)
	result.setStatus(StatusAdded)

	return result, nil
}

/**
* addEventJob
* @param id, name, spec, channel string, started bool, params et.Json, repetitions int
* @return *Job, error
**/
func (s *Jobs) addEventJob(id, name, spec, channel string, started bool, params et.Json, repetitions int) (*Job, error) {
	if !s.isServer {
		return nil, fmt.Errorf("crontab is not server")
	}

	result, err := s.addJob(id, name, spec, channel, params, repetitions, nil)
	if err != nil {
		return nil, err
	}

	if started {
		err = result.Start()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* indexJobById
* @param id string
* @return int
**/
func (s *Jobs) indexJobById(id string) int {
	return slices.IndexFunc(s.jobs, func(s *Job) bool { return s.Id == id })
}

/**
* indexJobByName
* @param name string
* @return int
**/
func (s *Jobs) indexJobByName(name string) int {
	return slices.IndexFunc(s.jobs, func(s *Job) bool { return s.Name == name })
}

/**
* removeJob
* @param job *Job
* @return error
**/
func (s *Jobs) removeJob(job *Job) error {
	if job.Started {
		job.Stop()
	}

	idx := s.indexJobById(job.Id)
	if idx == -1 {
		return nil
	}

	job.setStatus(StatusRemoved)

	return nil
}

/**
* deleteJobByName
* @param name string
* @return error
**/
func (s *Jobs) deleteJobByName(name string) error {
	idx := s.indexJobByName(name)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	return s.removeJob(job)
}

/**
* deleteJobById
* @param id string
* @return error
**/
func (s *Jobs) deleteJobById(id string) error {
	idx := s.indexJobById(id)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	return s.removeJob(job)
}

/**
* list
* @return et.Items
**/
func (s *Jobs) list() et.Items {
	var items = make([]et.Json, 0)
	for _, job := range s.jobs {
		items = append(items, job.ToJson())
	}

	return et.Items{
		Ok:     len(s.jobs) > 0,
		Count:  len(s.jobs),
		Result: items,
	}
}

/**
* stopJobByName
* @param name string
**/
func (s *Jobs) stopJobByName(name string) error {
	idx := s.indexJobByName(name)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	job.Stop()

	return nil
}

/**
* stopJobById
* @param id string
* @return error
**/
func (s *Jobs) stopJobById(id string) error {
	idx := s.indexJobById(id)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	job.Stop()

	return nil
}

/**
* startJobByName
* @param name string
* @return int, error
**/
func (s *Jobs) startJobByName(name string) (int, error) {
	idx := s.indexJobByName(name)
	if idx == -1 {
		return 0, fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	err := job.Start()
	if err != nil {
		return 0, err
	}

	return job.Idx, nil
}

/**
* startJobById
* @param id string
* @return error
**/
func (s *Jobs) startJobById(id string) error {
	idx := s.indexJobById(id)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	err := job.Start()
	if err != nil {
		return err
	}

	return nil
}

/**
* Start
* @return error
**/
func (s *Jobs) start() error {
	if s.crontab == nil {
		return fmt.Errorf("crontab not initialized")
	}

	if s.running {
		return nil
	}

	s.crontab.Start()
	s.running = true

	logs.Logf(packageName, `Crontab started`)

	return nil
}

/**
* stop
* @return error
**/
func (s *Jobs) stop() error {
	if s.crontab == nil {
		return fmt.Errorf("crontab not initialized")
	}

	s.crontab.Stop()
	s.running = false

	logs.Logf(packageName, `Crontab stopped`)

	return nil
}
