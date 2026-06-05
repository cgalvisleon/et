package crontab

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/robfig/cron/v3"
)

type JobStatus string

const (
	Pending  JobStatus = "pending"
	Awaiting JobStatus = "awaiting"
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
	ProjectId   string        `json:"project_id"`
	UserId      string        `json:"user_id"`
	ExecuteAt   time.Time     `json:"execute_at"`
	Type        TypeJob       `json:"type"`
	Tag         string        `json:"tag"`
	OwnerId     string        `json:"owner_id"`
	Channel     string        `json:"channel"`
	Params      et.Json       `json:"params"`
	Spec        string        `json:"spec"`
	Started     bool          `json:"started"`
	Status      JobStatus     `json:"status"`
	HostName    string        `json:"host_name"`
	Attempts    int           `json:"attempts"`
	Repetitions int           `json:"repetitions"`
	Duration    time.Duration `json:"duration"`
	isDebug     bool          `json:"-"`
	idx         cron.EntryID  `json:"-"`
	shot        *time.Timer   `json:"-"`
	owner       *Crontab      `json:"-"`
	mu          *sync.Mutex   `json:"-"`
	saveMu      sync.Mutex    `json:"-"`
	saveTimer   *time.Timer   `json:"-"`
}

/**
* newJob
* @param owner *Crontab, tp TypeJob, tag, ownerId, spec, channel string, params et.Json, repetitions int
* @return *Job
**/
func newJob(owner *Crontab, tp TypeJob, tag, ownerId, spec, channel string, params et.Json, repetitions int) *Job {
	id := reg.UUID()
	if ownerId == "" {
		ownerId = id
	}
	result := &Job{
		ID:          id,
		Type:        tp,
		Tag:         tag,
		OwnerId:     ownerId,
		Channel:     channel,
		Params:      params,
		Spec:        spec,
		Started:     false,
		Status:      Pending,
		HostName:    hostName,
		Attempts:    0,
		Repetitions: repetitions,
		idx:         -1,
		owner:       owner,
		isDebug:     owner.isDebug,
		mu:          &sync.Mutex{},
		saveMu:      sync.Mutex{},
	}

	return result
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Job) Serialize() ([]byte, error) {
	return utility.Serialize(s)
}

/**
* ToJson
* @return et.Json
**/
func (s *Job) ToJson() et.Json {
	return et.Json{
		"id":          s.ID,
		"project_id":  s.ProjectId,
		"user_id":     s.UserId,
		"execute_at":  timezone.Format(s.ExecuteAt, timezone.RFC3339),
		"type":        s.Type,
		"tag":         s.Tag,
		"owner_id":    s.OwnerId,
		"channel":     s.Channel,
		"params":      s.Params,
		"spec":        s.Spec,
		"started":     s.Started,
		"status":      s.Status,
		"host_name":   s.HostName,
		"attempts":    s.Attempts,
		"repetitions": s.Repetitions,
		"duration":    s.Duration,
	}
}

/**
* ToString
* @return string
**/
func (s *Job) ToString() string {
	return s.ToJson().ToString()
}

/**
* Save
* @return error
**/
func (s *Job) Save() error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}

	s.saveTimer = time.AfterFunc(100*time.Millisecond, func() {
		data := s.ToJson()
		if s.isDebug {
			logs.Log(packageName, "save:", data.ToString())
		}

		if s.owner.store != nil {
			err := s.owner.store.Set(s.ID, s.Tag, s.OwnerId, s.ProjectId, s, s.UserId)
			if err != nil {
				logs.Errorf("Error saving instance crontab: %v", err)
			}
		}

		event.Publish(EVENT_INSTANCE_SET, data)
	})
	return nil
}

/**
* setStatus
* @param status JobStatus
* @return error
**/
func (s *Job) setStatus(status JobStatus) error {
	if s.Status == status {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	logs.Logf(packageName, fmt.Sprintf("job:%s | status:%s | host:%s | attempt:%d | repetitions:%d", s.Tag, status, s.HostName, s.Attempts, s.Repetitions))
	return s.Save()
}

/**
* trigger
* @return void
**/
func (s *Job) trigger() {
	s.Attempts++
	err := event.Publish(s.Channel, s.Params)
	if err != nil {
		s.setStatus(Failed)
	} else {
		s.setStatus(Running)
	}
	if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
		s.Finish()
	} else if s.Type != CronJob {
		s.Finish()
	} else {
		s.setStatus(Awaiting)
	}
}

/**
* start
* @return error
**/
func (s *Job) start() error {
	if s.Type == CronJob {
		if s.idx != -1 {
			s.owner.cronJobs.Remove(s.idx)
		}

		idx, err := s.owner.cronJobs.AddFunc(s.Spec, s.trigger)
		if err != nil {
			return err
		}

		s.idx = idx
	} else {
		if s.shot != nil {
			s.shot.Stop()
		}

		now := timezone.Now()
		shotTime, err := timezone.Parse("2006-01-02T15:04:05", s.Spec)
		if err != nil {
			return err
		}
		if shotTime.After(now) {
			duration := shotTime.Sub(now)
			s.Duration = duration
			s.shot = time.AfterFunc(duration, s.trigger)
		}
	}

	return nil
}

/**
* stop
* @return void
**/
func (s *Job) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Type == CronJob {
		if s.idx != -1 {
			s.owner.cronJobs.Remove(s.idx)
			s.idx = -1
		}
	} else if s.shot != nil {
		s.shot.Stop()
	}
}

/**
* Start
* @return error
**/
func (s *Job) Start() error {
	s.Started = true
	time.AfterFunc(100*time.Millisecond, func() {
		s.start()
	})

	return s.setStatus(Awaiting)
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
	s.stop()
	s.setStatus(Awaiting)
}

/**
* Finish
* @return error
**/
func (s *Job) Finish() {
	s.Started = false
	s.stop()
	s.setStatus(Finished)
	time.AfterFunc(300*time.Millisecond, func() {
		delete(s.owner.Jobs, s.Tag)
	})
}

/**
* SetSpec
* @param spec string
* @return void
**/
func (s *Job) SetSpec(spec string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	isStarted := s.Started
	s.stop()
	s.Spec = spec
	if isStarted {
		s.start()
	}
}
