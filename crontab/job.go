package crontab

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/robfig/cron/v3"
)

type JobStatus string

const (
	Pending  JobStatus = "pending"
	Prepared JobStatus = "prepared"
	Running  JobStatus = "running"
	Done     JobStatus = "done"
	Failed   JobStatus = "failed"
	Finished JobStatus = "finished"
)

type TypeJob string

const (
	CronJob     TypeJob = "cronJob"
	ScheduleJob TypeJob = "scheduleJob"
)

type Job struct {
	ID          string        `json:"id"`
	ExecuteAt   time.Time     `json:"execute_at"`
	Type        TypeJob       `json:"type"`
	Tag         string        `json:"tag"`
	Channel     string        `json:"channel"`
	Params      et.Json       `json:"params"`
	Spec        string        `json:"spec"`
	Started     bool          `json:"started"`
	Status      JobStatus     `json:"status"`
	HostName    string        `json:"host_name"`
	Attempts    int           `json:"attempts"`
	Repetitions int           `json:"repetitions"`
	Duration    time.Duration `json:"duration"`
	idx         cron.EntryID  `json:"-"`
	shot        *time.Timer   `json:"-"`
	jobs        *Jobs         `json:"-"`
	mu          *sync.Mutex   `json:"-"`
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
* Save
* @return error
**/
func (s *Job) Save() error {
	if setInstance == nil {
		return nil
	}

	s.ID = utility.UUID()
	s.ExecuteAt = timezone.Now()
	return setInstance(s.ID, s.Tag, s)
}

/**
* setStatus
* @param status JobStatus
* @return void
**/
func (s *Job) setStatus(status JobStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	logs.Logf(packageName, fmt.Sprintf("Status: %s | job: %s | host: %s | attempt: %d", status, s.Tag, s.HostName, s.Attempts))
	go s.Save()
}

/**
* Start
* @return error
**/
func (s *Job) Start() error {
	fn := func() {
		s.Attempts++
		err := event.Publish(s.Channel, s.Params)
		if err != nil {
			s.setStatus(Failed)
		} else {
			s.setStatus(Running)
		}
		if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
			s.Finish()
		} else {
			s.setStatus(Pending)
		}
	}

	if s.Type == CronJob {
		id, err := s.jobs.cronJobs.AddFunc(s.Spec, fn)
		if err != nil {
			return err
		}

		s.idx = id
	} else {
		now := timezone.Now()
		shotTime, err := timezone.Parse("2006-01-02T15:04:05", s.Spec)
		if err != nil {
			return err
		}
		if shotTime.After(now) {
			duration := shotTime.Sub(now)
			s.Duration = duration
			s.shot = time.AfterFunc(duration, fn)
		} else if s.shot != nil {
			s.Stop()
		}
	}

	s.Started = true

	return s.Save()
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
	time.AfterFunc(1*time.Second, func() {
		if s.Type == CronJob {
			s.jobs.cronJobs.Remove(s.idx)
			s.idx = -1
		} else if s.shot != nil {
			s.shot.Stop()
		}
		s.setStatus(Pending)
	})
}

/**
* Finish
* @return error
**/
func (s *Job) Finish() {
	s.Stop()
	time.AfterFunc(1*time.Second, func() {
		s.setStatus(Finished)
	})
}
