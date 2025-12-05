package crontab

import (
	"fmt"
	"strings"

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
* @params tag string
* @return error
**/
func Load(tag string) error {
	if crontab != nil {
		return nil
	}

	tag = strings.ReplaceAll(tag, " ", "_")
	tag = strings.ToLower(tag)
	crontab = New()
	err := crontab.load()
	if err != nil {
		return err
	}

	err = crontab.start()
	if err != nil {
		return err
	}

	err = eventInit(tag)
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
* AddJob
* Add job to crontab in execute local
* @param tag, spec string, params et.Json, repetitions int, started bool, fn func(job *Job)
* @return *Job, error
**/
func AddJob(tag, spec string, params et.Json, repetitions int, started bool, fn func(job *Job)) (*Job, error) {
	if crontab == nil {
		return nil, fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	return crontab.addEventJob(TypeJobCron, tag, spec, "", started, params, repetitions, fn)
}

/**
* AddOneShotJob
* Add job to crontab in execute local
* @param tag, spec string, params et.Json, repetitions int, started bool, fn func(job *Job)
* @return *Job, error
**/
func AddOneShotJob(tag, spec string, params et.Json, repetitions int, started bool, fn func(job *Job)) (*Job, error) {
	if crontab == nil {
		return nil, fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	return crontab.addEventJob(TypeJobOneShot, tag, spec, "", started, params, repetitions, fn)
}

/**
* AddEventJob
* Event job to crontab function execute was notified by event workers
* @param tag, spec, channel string, repetitions int, started bool, params et.Json, fn func(event.Message)
* @return *Job, error
**/
func AddEventJob(tag, spec, channel string, repetitions int, started bool, params et.Json, fn func(event.Message)) error {
	if crontab == nil {
		return fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	event.Publish(EVENT_CRONTAB_SET, et.Json{
		"type":        TypeJobCron,
		"tag":         tag,
		"spec":        spec,
		"channel":     channel,
		"repetitions": repetitions,
		"started":     started,
		"params":      params,
	})

	err := event.Stack(channel, fn)
	if err != nil {
		return err
	}

	return nil
}

/**
* AddOneShotEventJob
* Event job to crontab function execute was notified by event workers
* @param tag, spec, channel string, repetitions int, started bool, params et.Json, fn func(event.Message)
* @return *Job, error
**/
func AddOneShotEventJob(tag, spec, channel string, started bool, params et.Json, fn func(event.Message)) error {
	if crontab == nil {
		return fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	event.Publish(EVENT_CRONTAB_SET, et.Json{
		"type":    TypeJobOneShot,
		"tag":     tag,
		"spec":    spec,
		"channel": channel,
		"started": started,
		"params":  params,
	})

	err := event.Stack(channel, fn)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteJob
* @param tag string
* @return error
**/
func DeleteJob(tag string) error {
	if crontab == nil {
		return fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_DELETE, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* StartJob
* @param tag string
* @return int, error
**/
func StartJob(tag string) (int, error) {
	if crontab == nil {
		return 0, fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_START, et.Json{"tag": tag})
	if err != nil {
		return 0, err
	}

	return 1, nil
}

/**
* StopJob
* @param tag string
* @return error
**/
func StopJob(tag string) error {
	if crontab == nil {
		return fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_STOP, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* Stop
* @return error
**/
func Stop() error {
	if crontab == nil {
		return fmt.Errorf(MSG_CRONTAB_UNLOAD)
	}

	return crontab.stop()
}
