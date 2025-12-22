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
	StatusPending  JobStatus = "pending"
	StatusRunning  JobStatus = "running"
	StatusDone     JobStatus = "done"
	StatusFailed   JobStatus = "failed"
	StatusStop     JobStatus = "stop"
	StatusFinished JobStatus = "finished"
)

type TypeJob string

const (
	TypeCronJob TypeJob = "cron-job"
	TypeOneShot TypeJob = "one-shot"
)

type Job struct {
	Type        TypeJob       `json:"type"`
	Tag         string        `json:"tag"`
	Channel     string        `json:"channel"`
	Params      et.Json       `json:"params"`
	Spec        string        `json:"spec"`
	Started     bool          `json:"started"`
	Status      JobStatus     `json:"status"`
	Idx         int           `json:"idx"`
	HostName    string        `json:"host_name"`
	Attempts    int           `json:"attempts"`
	Repetitions int           `json:"repetitions"`
	ShotTime    time.Time     `json:"shot_time"`
	Duration    time.Duration `json:"duration"`
	jobs        *Jobs         `json:"-"`
	mu          *sync.Mutex   `json:"-"`
	shot        *time.Timer   `json:"-"`
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
func (s *Job) save() {
	logs.Logf(packageName, fmt.Sprintf("Job %s status:%s host:%s attempt:%d", s.Tag, s.Status, s.HostName, s.Attempts))
	event.Publish(EVENT_CRONTAB_STATUS, s.ToJson())
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
	go s.save()
}

/**
* Start
* @return error
**/
func (s *Job) Start() error {
	if s.Started {
		return nil
	}

	fn := func() {
		s.setStatus(StatusRunning)
		s.Attempts++
		err := event.Publish(s.Channel, s.Params)
		if err != nil {
			s.setStatus(StatusFailed)
		}
		if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
			s.Stop()
		} else {
			s.setStatus(StatusPending)
		}
	}

	if s.Type == TypeCronJob {
		id, err := s.jobs.crontab.AddFunc(s.Spec, fn)
		if err != nil {
			return err
		}

		s.Idx = int(id)
	} else {
		now := timezone.NowTime()
		if s.ShotTime.After(now) {
			s.Duration = s.ShotTime.Sub(now)
			s.shot = time.AfterFunc(s.Duration, fn)
		}
	}

	s.Started = true
	s.setStatus(StatusPending)

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
		if s.Type == TypeCronJob {
			s.jobs.crontab.Remove(cron.EntryID(s.Idx))
			s.Idx = -1
		} else if s.shot != nil {
			s.shot.Stop()
		}
		s.setStatus(StatusStop)
	})
}
