package crontab

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/robfig/cron/v3"
)

type JobStatus string

const (
	Pending  JobStatus = "pending"
	Running  JobStatus = "running"
	Done     JobStatus = "done"
	Failed   JobStatus = "failed"
	Finished JobStatus = "finished"
	Awaiting JobStatus = "awaiting"
)

type TypeJob string

const (
	CRONJOB     TypeJob = "cronJob"
	SCHEDULEJOB TypeJob = "scheduleJob"
)

type Job struct {
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	TenantId    string        `json:"tenant_id"`
	ID          string        `json:"id"`
	Type        TypeJob       `json:"type"`
	Tag         string        `json:"tag"`
	OwnerId     string        `json:"owner_id"`
	Spec        string        `json:"spec"`
	Params      et.Json       `json:"params"`
	Status      JobStatus     `json:"status"`
	HostName    string        `json:"host_name"`
	Repetitions int           `json:"repetitions"`
	Attempts    int           `json:"attempts"`
	Duration    time.Duration `json:"duration"`
	AuditLog    []et.Json     `json:"audit_log"`
	isDebug     bool          `json:"-"`
	isChanged   bool          `json:"-"`
	idx         cron.EntryID  `json:"-"`
	shot        *time.Timer   `json:"-"`
	crontab     *Crontab      `json:"-"`
	store       Store         `json:"-"`
	mu          *sync.Mutex   `json:"-"`
}

/**
* newJob
* @param tp TypeJob, tag, ownerId, spec string, params et.Json, repetitions int
* @return *Job
**/
func newJob(tenantId string, tp TypeJob, tag, ownerId, spec string, params et.Json, repetitions int) *Job {
	now := timezone.Now()
	id := reg.ULID()
	result := &Job{
		CreatedAt:   now,
		UpdatedAt:   now,
		TenantId:    tenantId,
		ID:          id,
		Type:        tp,
		Tag:         tag,
		OwnerId:     ownerId,
		Spec:        spec,
		Params:      params,
		Status:      Pending,
		Repetitions: repetitions,
		idx:         -1,
	}
	event.Publish(EVENT_CRONTAB_SET, result.ToJson())
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Job) ToJson() et.Json {
	return et.Json{
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":   s.TenantId,
		"id":          s.ID,
		"type":        s.Type,
		"tag":         s.Tag,
		"owner_id":    s.OwnerId,
		"spec":        s.Spec,
		"params":      s.Params,
		"status":      s.Status,
		"host_name":   s.HostName,
		"repetitions": s.Repetitions,
		"attempts":    s.Attempts,
		"duration":    s.Duration,
		"audit_log":   s.AuditLog,
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
* up
* @param crontab *Crontab
* @return error
**/
func (s *Job) up(crontab *Crontab) error {
	s.HostName = crontab.HostName
	s.crontab = crontab
	s.store = crontab.store
	s.mu = &sync.Mutex{}
	s.isDebug = crontab.isDebug
	return s.save()
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Job) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
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
* save
* @return error
**/
func (s *Job) save() error {
	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	if s.store == nil {
		return nil
	}

	err := s.store.Set("job", s.ID, s.TenantId, s.OwnerId, s)
	if err != nil {
		return err
	}
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

	s.addAuditLog(s.HostName, string(status))
	logs.Logf(packageName, MSG_JOB_STATUS, s.Tag, status, s.HostName, s.Attempts, s.Repetitions)

	return s.save()
}

/**
* trigger
* @return void
**/
func (s *Job) trigger() {
	s.setStatus(Running)
	s.Attempts++
	channel := fmt.Sprintf("job:%s:%s", s.TenantId, s.Tag)
	err := event.Publish(channel, s.Params)
	if err != nil {
		s.setStatus(Failed)
	}

	if s.Repetitions != 0 && s.Attempts >= s.Repetitions {
		s.finish()
	} else if s.Type != CRONJOB {
		s.finish()
	} else {
		s.setStatus(Awaiting)
	}
}

/**
* start
* @return error
**/
func (s *Job) start() error {
	if s.Type == CRONJOB {
		if s.idx != -1 {
			s.crontab.cronJobs.Remove(s.idx)
		}

		idx, err := s.crontab.cronJobs.AddFunc(s.Spec, s.trigger)
		if err != nil {
			return err
		}

		s.idx = idx
		return s.save()
	}

	if s.shot != nil {
		s.shot.Stop()
	}

	now := timezone.Now()
	shotTime, err := timezone.Parse(time.RFC3339, s.Spec)
	if err != nil {
		return err
	}

	if shotTime.After(now) {
		duration := shotTime.Sub(now)
		s.Duration = duration
		s.shot = time.AfterFunc(duration, s.trigger)
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

	if s.Type == CRONJOB {
		if s.idx != -1 {
			s.crontab.cronJobs.Remove(s.idx)
			s.idx = -1
		}
	} else if s.shot != nil {
		s.shot.Stop()
	}
}

/**
* finish
* @return void
**/
func (s *Job) finish() {
	s.stop()
	s.setStatus(Finished)
}
