package crontab

import (
	"errors"

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

	var err error
	crontab, err = newCrontab(tag)
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
* Add
* Add job to crontab in execute local
* @param tp TypeJob, tag, spec string, params et.Json, repetitions int, fn func(event.Message)
* @return error
**/
func Add(tp TypeJob, tag, spec string, params et.Json, repetitions int, fn func(event.Message)) error {
	if crontab == nil {
		return errors.New(MSG_CRONTAB_UNLOAD)
	}

	event.Publish(EVENT_CRONTAB_SET, et.Json{
		"type":        tp,
		"tag":         tag,
		"spec":        spec,
		"channel":     tag,
		"repetitions": repetitions,
		"params":      params,
	})

	err := event.Stack(tag, fn)
	if err != nil {
		return err
	}

	return nil
}

/**
* AddCronjob
* @param tag, spec string, params et.Json, repetitions int, fn func(event.Message)
* @return error
**/
func AddCronjob(tag, spec string, params et.Json, repetitions int, fn func(event.Message)) error {
	return Add(TypeCronJob, tag, spec, params, repetitions, fn)
}

/**
* AddOneShot
* @param tag, spec string, params et.Json, repetitions int, fn func(event.Message)
* @return error
**/
func AddOneShot(tag, spec string, params et.Json, repetitions int, fn func(event.Message)) error {
	return Add(TypeOneShot, tag, spec, params, repetitions, fn)
}

/**
* Remove
* @param tag string
* @return error
**/
func Remove(tag string) error {
	if crontab == nil {
		return errors.New(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_REMOVE, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* Start
* @param tag string
* @return int, error
**/
func Start(tag string) (int, error) {
	if crontab == nil {
		return 0, errors.New(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_START, et.Json{"tag": tag})
	if err != nil {
		return 0, err
	}

	return 1, nil
}

/**
* Stop
* @param tag string
* @return error
**/
func Stop(tag string) error {
	if crontab == nil {
		return errors.New(MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_STOP, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* StartCrontab
* @return error
**/
func StartCrontab() error {
	if crontab == nil {
		return errors.New(MSG_CRONTAB_UNLOAD)
	}

	return crontab.start()
}

/**
* StopCrontab
* @return error
**/
func StopCrontab() error {
	if crontab == nil {
		return errors.New(MSG_CRONTAB_UNLOAD)
	}

	return crontab.stop()
}
