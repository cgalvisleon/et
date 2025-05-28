package crontab

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/robfig/cron/v3"
)

type Job struct {
	Id      string  `json:"id"`
	Name    string  `json:"name"`
	Channel string  `json:"channel"`
	Params  et.Json `json:"params"`
	Spec    string  `json:"spec"`
	Started bool    `json:"started"`
	Idx     int     `json:"idx"`
	fn      func()  `json:"-"`
}

type Jobs struct {
	Id      string     `json:"id"`
	Started bool       `json:"started"`
	jobs    []*Job     `json:"-"`
	crontab *cron.Cron `json:"-"`
}

func New() *Jobs {
	return &Jobs{
		Id:      utility.UUID(),
		jobs:    make([]*Job, 0),
		crontab: cron.New(),
	}
}

/**
* removeJob
* @param idx int
* @return error
**/
func (s *Jobs) removeJob(idx int) error {
	job := s.jobs[idx]
	if job.Started {
		time.AfterFunc(time.Second*3, func() {
			id := job.Idx
			s.crontab.Remove(cron.EntryID(id))
		})
	}

	s.jobs = slices.Delete(s.jobs, idx, idx+1)

	return nil
}

/**
* startJob
* @param idx int
* @return int, error
**/
func (s *Jobs) startJob(idx int) (int, error) {
	job := s.jobs[idx]
	if job.Started {
		return 0, errors.New("job already started")
	}

	if job.fn == nil {
		job.fn = func() {
			event.Publish(job.Channel, job.Params)
		}
	}

	id, err := s.crontab.AddFunc(job.Spec, job.fn)
	if err != nil {
		return 0, err
	}

	job.Idx = int(id)
	job.Started = true

	return job.Idx, nil
}

/**
* stopJobs
* @param idx int
* @return error
**/
func (s *Jobs) stopJobs(idx int) error {
	job := s.jobs[idx]
	if !job.Started {
		return errors.New("job not started")
	}

	time.AfterFunc(time.Second*3, func() {
		s.crontab.Remove(cron.EntryID(job.Idx))
	})

	job.Started = false
	job.Idx = -1

	return nil
}

/**
* AddJob
* @param id, name, spec, channel string, params et.Json
* @return int, error
**/
func (s *Jobs) AddJob(id, name, spec, channel string, params et.Json, fn func()) error {
	if !utility.ValidStr(name, 0, []string{"", " "}) {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Name == name })
	if idx != -1 {
		return errors.New("job already exists")
	}

	s.jobs = append(s.jobs, &Job{
		Id:      id,
		Name:    name,
		Channel: channel,
		Params:  params,
		Spec:    spec,
		Started: false,
		Idx:     len(s.jobs),
		fn:      fn,
	})

	return nil
}

/**
* DeleteJob
* @param name string
* @return error
**/
func (s *Jobs) DeleteJob(name string) error {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Name == name })
	if idx == -1 {
		return errors.New("job not found")
	}

	return s.removeJob(idx)
}

/**
* DeleteJobById
* @param id string
* @return error
**/
func (s *Jobs) DeleteJobById(id string) error {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Id == id })
	if idx == -1 {
		return errors.New("job not found")
	}

	return s.removeJob(idx)
}

/**
* List
* @return et.Items
**/
func (s *Jobs) List() et.Items {
	var items = make([]et.Json, 0)
	for _, job := range s.jobs {
		items = append(items, et.Json{
			"idx":     job.Idx,
			"name":    job.Name,
			"spec":    job.Spec,
			"started": job.Started,
		})
	}

	return et.Items{
		Ok:     len(s.jobs) > 0,
		Count:  len(s.jobs),
		Result: items,
	}
}

/**
* StartJob
* @param name string
* @return int, error
**/
func (s *Jobs) StartJob(name string) (int, error) {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Name == name })
	if idx == -1 {
		return 0, errors.New("job not found")
	}

	return s.startJob(idx)
}

/**
* StartJobById
* @param id string
* @return int, error
**/
func (s *Jobs) StartJobById(id string) (int, error) {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Id == id })
	if idx == -1 {
		return 0, errors.New("job not found")
	}

	return s.startJob(idx)
}

/**
* StopJob
* @param name string
**/
func (s *Jobs) StopJob(name string) error {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Name == name })
	if idx == -1 {
		return errors.New("job not found")
	}

	return s.stopJobs(idx)
}

/**
* StopJobById
* @param id string
* @return error
**/
func (s *Jobs) StopJobById(id string) error {
	idx := slices.IndexFunc(s.jobs, func(j *Job) bool { return j.Id == id })
	if idx == -1 {
		return errors.New("job not found")
	}

	return s.stopJobs(idx)
}

/**
* Start
* @return error
**/
func (s *Jobs) Start() error {
	if s.crontab == nil {
		return errors.New("crontab not initialized")
	}

	s.crontab.Start()

	return nil
}

/**
* Stop
* @return error
**/
func (s *Jobs) Stop() error {
	if s.crontab == nil {
		return errors.New("crontab not initialized")
	}

	s.crontab.Stop()

	return nil
}
