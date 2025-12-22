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
	"github.com/cgalvisleon/et/reg"
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
	Tag     string     `json:"tag"`
	Version int        `json:"version"`
	jobs    []*Job     `json:"-"`
	crontab *cron.Cron `json:"-"`
}

/**
* newCrontab
* @param tag string
* @return *Jobs, error
**/
func newCrontab(tag string) (*Jobs, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	if tag == "" {
		tag = reg.GenULID(packageName)
	}

	return &Jobs{
		Tag:     tag,
		jobs:    make([]*Job, 0),
		crontab: cron.New(cron.WithSeconds()),
	}, nil
}

/**
* indexJob
* @param tag string
* @return int
**/
func (s *Jobs) indexJob(tag string) int {
	return slices.IndexFunc(s.jobs, func(s *Job) bool { return s.Tag == tag })
}

/**
* addJob
* @param tp TypeJob, tag, spec, channel string, params et.Json, repetitions int
* @return *Job, error
**/
func (s *Jobs) addJob(tp TypeJob, tag, spec, channel string, params et.Json, repetitions int) (*Job, error) {
	if !utility.ValidStr(tag, 0, []string{"", " "}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}

	shot, err := timezone.Parse("2006-01-02T15:04:05", spec)
	if err != nil {
		shot = timezone.NowTime()
	}

	idx := s.indexJob(tag)
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
		HostName:    hostName,
		Repetitions: repetitions,
		jobs:        s,
		mu:          &sync.Mutex{},
	}
	s.jobs = append(s.jobs, result)
	result.setStatus(StatusPending)

	err = result.Start()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* removeJob
* @param idx int
* @return error
**/
func (s *Jobs) removeJob(tag string) error {
	idx := s.indexJob(tag)
	if idx == -1 {
		return nil
	}

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
* stopJob
* @param tag string
* @return error
**/
func (s *Jobs) stopJob(tag string) error {
	idx := s.indexJob(tag)
	if idx == -1 {
		return nil
	}

	job := s.jobs[idx]
	job.Stop()

	return nil
}

/**
* startJob
* @param tag string
* @return int, error
**/
func (s *Jobs) startJob(tag string) error {
	idx := s.indexJob(tag)
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

	for _, job := range s.jobs {
		job.Start()
	}
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

	for _, job := range s.jobs {
		job.Stop()
	}
	logs.Logf(packageName, `Crontab stopped`)

	return nil
}
