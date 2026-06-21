package crontab

import (
	"fmt"
	"os"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/robfig/cron/v3"
)

const (
	packageName = "crontab"
)

var (
	hostName, _  = os.Hostname()
	ErrJobExists = fmt.Errorf("job already exists")
)

type Store interface {
	Set(collection, id, tenantId, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}

type Crontab struct {
	TenantId string          `json:"tenant_id"`
	HostName string          `json:"host_name"`
	Jobs     map[string]*Job `json:"jobs"`
	cronJobs *cron.Cron      `json:"-"`
	running  bool            `json:"-"`
	mu       *sync.Mutex     `json:"-"`
	store    Store           `json:"-"`
	isDebug  bool            `json:"-"`
}

/**
* New
* @param tenantId string, store Store
* @return (*Crontab, error)
**/
func New(tenantId string, store Store) (*Crontab, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	loc := timezone.Location()
	result := &Crontab{
		TenantId: tenantId,
		HostName: hostName,
		Jobs:     make(map[string]*Job),
		cronJobs: cron.New(
			cron.WithSeconds(),
			cron.WithLocation(loc),
		),
		mu:    &sync.Mutex{},
		store: store,
	}
	result.cronJobs.Start()

	return result, nil
}

/**
* Debug
* @return *Crontab
**/
func (s *Crontab) Debug() *Crontab {
	s.isDebug = true
	return s
}

/**
* addJob
* @param tp TypeJob, tag, ownerId, spec, channel string, started bool, params et.Json, repetitions int
* @return *Job, error
**/
func (s *Crontab) addJob(job *Job) error {
	s.mu.Lock()
	_, exists := s.Jobs[job.ID]
	s.mu.Unlock()
	if exists {
		return nil
	}

	s.mu.Lock()
	s.Jobs[job.ID] = job
	s.mu.Unlock()

	err := job.up(s)
	if err != nil {
		return err
	}

	err = job.start()
	if err != nil {
		return err
	}

	logs.Log(packageName, fmt.Sprintf(MSG_ADD_JOB, job.ID, job.Tag, job.Type, job.Spec))

	return nil
}

/**
* removeJob
* @param tag string
* @return bool
**/
func (s *Crontab) removeJob(id string) bool {
	s.mu.Lock()
	job, exists := s.Jobs[id]
	s.mu.Unlock()
	if !exists {
		return false
	}

	job.stop()

	s.mu.Lock()
	delete(s.Jobs, id)
	s.mu.Unlock()

	logs.Log(packageName, fmt.Sprintf(MSG_REMOVE_JOB, id))
	return true
}

/**
* startJob
* @param tag string
* @return error
**/
func (s *Crontab) startJob(id string) error {
	s.mu.Lock()
	job, exists := s.Jobs[id]
	s.mu.Unlock()
	if !exists {
		return fmt.Errorf("job not found")
	}

	err := job.start()
	if err != nil {
		return err
	}

	return nil
}

/**
* stopJob
* @param tag string
* @return error
**/
func (s *Crontab) stopJob(tag string) error {
	s.mu.Lock()
	job, exists := s.Jobs[tag]
	s.mu.Unlock()
	if !exists {
		return fmt.Errorf("job not found")
	}

	job.stop()
	return nil
}
