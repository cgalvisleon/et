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
	"github.com/robfig/cron/v3"
)

type JobStatus string

const (
	JobStatusPending  JobStatus = "pending"
	JobStatusRunning  JobStatus = "running"
	JobStatusDone     JobStatus = "done"
	JobStatusFailed   JobStatus = "failed"
	JobStatusStop     JobStatus = "stop"
	JobStatusFinished JobStatus = "finished"
)

type TypeJob string

const (
	TypeJobCron    TypeJob = "cron"
	TypeJobOneShot TypeJob = "one-shot"
)

type Job struct {
	Type        TypeJob        `json:"type"`
	Tag         string         `json:"tag"`
	Channel     string         `json:"channel"`
	Params      et.Json        `json:"params"`
	Spec        string         `json:"spec"`
	Started     bool           `json:"started"`
	Status      JobStatus      `json:"status"`
	Idx         int            `json:"idx"`
	HostName    string         `json:"host_name"`
	Attempts    int            `json:"attempts"`
	Repetitions int            `json:"repetitions"`
	Duration    time.Duration  `json:"duration"`
	ShotTime    time.Time      `json:"shot_time"`
	fn          func(job *Job) `json:"-"`
	shot        *time.Timer    `json:"-"`
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
* save
* @return error
**/
func (s *Job) save() error {
	if set == nil {
		return nil
	}

	return set(s)
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
	logs.Logf(packageName, fmt.Sprintf("Job %s status:%s host:%s attempt:%d", s.Tag, s.Status, s.HostName, s.Attempts))
	go s.save()
}

/**
* Start
* @return error
**/
func (s *Job) Start() error {
	if s.fn == nil {
		s.fn = func(job *Job) {
			err := event.Publish(job.Channel, job.Params)
			if err != nil {
				s.setStatus(JobStatusFailed)
			}
		}
	}

	fn := func() {
		if s.fn != nil && s.Started {
			s.setStatus(JobStatusRunning)
			s.Attempts++
			s.fn(s)
			if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
				s.Remove()
			} else {
				s.setStatus(JobStatusPending)
			}
		}
	}

	if s.Started {
		return nil
	}

	if s.Type == TypeJobCron {
		id, err := s.jobs.crontab.AddFunc(s.Spec, fn)
		if err != nil {
			return err
		}

		s.Idx = int(id)
	} else {
		now := timezone.NowTime()
		shotTime := s.ShotTime
		if shotTime.After(now) {
			duration := s.ShotTime.Sub(now)
			s.Duration = duration
			s.shot = time.AfterFunc(duration, fn)
		} else if s.shot != nil {
			s.shot.Stop()
		}
	}

	s.Started = true

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
	time.AfterFunc(time.Second*1, func() {
		if s.Type == TypeJobCron {
			s.jobs.crontab.Remove(cron.EntryID(s.Idx))
			s.Idx = -1
		} else if s.shot != nil {
			s.shot.Stop()
		}
		s.setStatus(JobStatusStop)
	})
}

/**
* Remove
* @return error
**/
func (s *Job) Remove() {
	s.setStatus(JobStatusFinished)
	idx := s.jobs.indexJobByTag(s.Tag)
	if idx == -1 {
		return
	}

	s.jobs.removeJob(idx)
}
