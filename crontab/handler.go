package crontab

import (
	"fmt"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	crontab *Jobs
)

/**
* Load
**/
func Load() error {
	if crontab != nil {
		return nil
	}

	crontab = New()
	err := crontab.load()
	if err != nil {
		return err
	}

	return crontab.start()
}

/**
* Server
* @return error
**/
func Server() error {
	if crontab != nil {
		return nil
	}

	crontab = New()
	crontab.isServer = true
	err := crontab.load()
	if err != nil {
		return err
	}

	err = crontab.start()
	if err != nil {
		return err
	}

	err = eventInit()
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	if crontab == nil {
		return
	}

	cache.Delete("crontab:nodes")

	logs.Log(packageName, `Disconnect...`)
}

/**
* IsMaster
* @return bool
**/
func IsMaster() bool {
	return crontab.nodeId == 1
}

/**
* AddJob
* Add job to crontab in execute local
* @param id, name, spec, channel string, params et.Json, repetitions int, start bool, fn func()
* @return *Job, error
**/
func AddJob(id, name, spec, channel string, params et.Json, repetitions int, start bool, fn func(job *Job)) (*Job, error) {
	err := Load()
	if err != nil {
		return nil, err
	}

	if crontab.isServer {
		return nil, fmt.Errorf("crontab is server")
	}

	result, err := crontab.addJob(id, name, spec, channel, params, repetitions, fn)
	if err != nil {
		return nil, err
	}

	if !start {
		return result, nil
	}

	err = result.Start()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* PushEventJob
* Push job to crontab was notified by event workers
* @param id, name, spec, channel string, repetitions int, start bool, params et.Json
* @return error
**/
func PushEventJob(id, name, spec, channel string, repetitions int, start bool, params et.Json) error {
	err := Server()
	if err != nil {
		return err
	}

	return event.Publish(EVENT_CRONTAB_SET, et.Json{
		"id":          id,
		"name":        name,
		"spec":        spec,
		"channel":     channel,
		"repetitions": repetitions,
		"start":       start,
		"params":      params,
	})
}

/**
* EventJob
* Event job to crontab function execute was notified by event workers
* @param id, name, spec, channel string, repetitions int, start bool, params et.Json, fn func(event.Message)
* @return *Job, error
**/
func EventJob(id, name, spec, channel string, repetitions int, start bool, params et.Json, fn func(event.Message)) error {
	event.Publish(EVENT_CRONTAB_SET, et.Json{
		"id":          id,
		"name":        name,
		"spec":        spec,
		"channel":     channel,
		"repetitions": repetitions,
		"start":       start,
		"params":      params,
	})

	err := event.Stack(channel, fn)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteJob
* @param name string
* @return error
**/
func DeleteJob(name string) error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.deleteJobByName(name)
}

/**
* DeleteJobById
* @param id string
* @return error
**/
func DeleteJobById(id string) error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.deleteJobById(id)
}

/**
* StartJob
* @param name string
* @return int, error
**/
func StartJob(name string) (int, error) {
	err := Load()
	if err != nil {
		return 0, err
	}

	return crontab.startJobByName(name)
}

/**
* StartJobById
* @param id string
* @return error
**/
func StartJobById(id string) error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.startJobById(id)
}

/**
* StopJob
* @param name string
* @return error
**/
func StopJob(name string) error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.stopJobByName(name)
}

/**
* StopJobById
* @param id string
* @return error
**/
func StopJobById(id string) error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.stopJobById(id)
}

/**
* ListJobs
* @return et.Items, error
**/
func ListJobs() (et.Items, error) {
	err := Load()
	if err != nil {
		return et.Items{}, err
	}

	return crontab.list(), nil
}

/**
* Start
* @return error
**/
func Start() error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.start()
}

/**
* Stop
* @return error
**/
func Stop() error {
	err := Load()
	if err != nil {
		return err
	}

	return crontab.stop()
}

/**
* EventStatusRunning
* @param data et.Json
* @return error
**/
func EventStatusRunning(data et.Json) error {
	data.Set("status", StatusRunning)
	return event.Publish(EVENT_CRONTAB_STATUS, data)
}

/**
* EventStatusPending
* @param data et.Json
* @return error
**/
func EventStatusDone(data et.Json) error {
	data.Set("status", StatusDone)
	return event.Publish(EVENT_CRONTAB_STATUS, data)
}

/**
* EventStatusFailed
* @param data et.Json
* @return error
**/
func EventStatusFailed(data et.Json) error {
	data.Set("status", StatusFailed)
	return event.Publish(EVENT_CRONTAB_STATUS, data)
}
