package crontab

import (
	"fmt"
	"os"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/robfig/cron/v3"
)

const (
	packageName = "crontab"
)

var (
	hostName, _  = os.Hostname()
	ErrJobExists = fmt.Errorf("job already exists")
)

type Jobs struct {
	Id         string     `json:"id"`
	HostName   string     `json:"host_name"`
	jobs       []*Job     `json:"-"`
	crontab    *cron.Cron `json:"-"`
	storageKey string     `json:"-"`
	running    bool       `json:"-"`
}

func New() *Jobs {
	version := "v0.0.1"
	return &Jobs{
		Id:         utility.UUID(),
		HostName:   hostName,
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
	err := event.Load()
	if err != nil {
		return err
	}

	return nil
}

/**
* addJob
* @param tp TypeJob, tag, spec, channel string, params et.Json, repetitions int, fn func(job *Job)
* @return *Job, error
**/
func (s *Jobs) addJob(tp TypeJob, tag, spec, channel string, params et.Json, repetitions int, fn func(job *Job)) (*Job, error) {
	if !utility.ValidStr(tag, 0, []string{"", " "}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}

	shot, err := timezone.Parse("2006-01-02T15:04:05", spec)
	if err != nil {
		shot = timezone.NowTime()
	}

	idx := s.indexJobByTag(tag)
	if idx != -1 {
		result := s.jobs[idx]
		if result.Spec != spec {
			result.Spec = spec
			result.ShotTime = shot
			result.Stop()
		}

		return result, nil
	}

	result := &Job{
		Type:        tp,
		Tag:         tag,
		Channel:     channel,
		Params:      params,
		Spec:        spec,
		ShotTime:    shot,
		Started:     false,
		Idx:         len(s.jobs),
		Repetitions: repetitions,
		fn:          fn,
		jobs:        s,
		mu:          &sync.Mutex{},
	}
	s.jobs = append(s.jobs, result)
	result.setStatus(JobStatusPending)

	return result, nil
}

/**
* addEventJob
* @param tp TypeJob, tag, spec, channel string, started bool, params et.Json, repetitions int
* @return *Job, error
**/
func (s *Jobs) addEventJob(tp TypeJob, tag, spec, channel string, started bool, params et.Json, repetitions int, fn func(job *Job)) (*Job, error) {
	result, err := s.addJob(tp, tag, spec, channel, params, repetitions, fn)
	if err != nil {
		return nil, err
	}

	if !started {
		return result, nil
	}

	err = result.Start()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* indexJobByTag
* @param tag string
* @return int
**/
func (s *Jobs) indexJobByTag(tag string) int {
	return slices.IndexFunc(s.jobs, func(s *Job) bool { return s.Tag == tag })
}

/**
* removeJob
* @param idx int
* @return error
**/
func (s *Jobs) removeJob(idx int) error {
	job := s.jobs[idx]
	if job == nil {
		return nil
	}

	if job.Started {
		job.Stop()
	}

	s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)

	return nil
}

/**
* deleteJobByTag
* @param tag string
* @return error
**/
func (s *Jobs) deleteJobByTag(tag string) error {
	idx := s.indexJobByTag(tag)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	err := s.removeJob(idx)
	if err != nil {
		return err
	}

	if delete != nil {
		return delete(tag)
	}

	return nil
}

/**
* stopJobByTag
* @param tag string
* @return error
**/
func (s *Jobs) stopJobByTag(tag string) error {
	idx := s.indexJobByTag(tag)
	if idx == -1 {
		return fmt.Errorf("job not found")
	}

	job := s.jobs[idx]
	job.Stop()

	return nil
}

/**
* startJobByTag
* @param tag string
* @return int, error
**/
func (s *Jobs) startJobByTag(tag string) (int, error) {
	idx := s.indexJobByTag(tag)
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
