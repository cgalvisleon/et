package crontab

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

var crontab *Jobs

func init() {
	crontab = New("crontab")
	err := cache.Load()
	if err != nil {
		panic(err)
	}

	err = event.Load()
	if err != nil {
		panic(err)
	}
}

/**
* Load
* @params db *jdb.DB, schemaName, tag string
* @return error
**/
func Load(tag string, store instances.Store) error {
	tag = strs.Name(tag)
	crontab = New(tag)
	err := crontab.start()
	if err != nil {
		return err
	}

	if store != nil {
		SetGetInstance(store.Get)
		SetSetInstance(store.Set)
	}

	return nil
}

/**
* Close
* @return void
**/
func Close() {
	if crontab == nil {
		return
	}

	cache.Delete("crontab:nodes")

	logs.Log(packageName, `Disconnect...`)
}

/**
* addJob
* @param jobType TypeJob, tag, spec, channel string, started bool, params et.Json, repetitions int, fn func(event.Message)
* @return error
**/
func addJob(jobType TypeJob, tag, spec, channel string, started bool, params et.Json, repetitions int, fn func(event.Message)) error {
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	tag = strs.Name(tag)
	data := et.Json{
		"type":        jobType,
		"tag":         tag,
		"spec":        spec,
		"channel":     channel,
		"started":     started,
		"params":      params,
		"repetitions": repetitions,
	}

	event.Publish(EVENT_CRONTAB_SET, data)
	err := event.Stack(channel, fn)
	if err != nil {
		return err
	}

	logs.Logf(packageName, "Add EventJob: %s", data.ToString())

	return nil
}

/**
* AddEventJob
* @param tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)
* @return error
**/
func AddEventJob(tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)) error {
	tag = strs.Name(tag)
	channel := fmt.Sprintf("cronjob:%s", tag)
	return addJob(CronJob, tag, spec, channel, started, params, repetitions, fn)
}

/**
* AddCronJob
* @param tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)
* @return error
**/
func AddCronJob(tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)) error {
	return AddEventJob(tag, spec, repetitions, started, params, fn)
}

/**
* AddScheduleJob
* Add job to crontab in execute local
* @param tag, schedule string, started bool, params et.Json, fn func(event.Message)
* @return error
**/
func AddScheduleJob(tag, schedule string, started bool, params et.Json, fn func(event.Message)) error {
	tag = strs.Name(tag)
	channel := fmt.Sprintf("schedule:%s", tag)
	return addJob(ScheduleJob, tag, schedule, channel, started, params, 0, fn)
}

/**
* RemoveJob
* @param tag string
* @return error
**/
func RemoveJob(tag string) error {
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_REMOVE, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* StartJob
* @param tag string
* @return error
**/
func StartJob(tag string) error {
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	err := event.Publish(EVENT_CRONTAB_START, et.Json{"tag": tag})
	if err != nil {
		return err
	}

	return nil
}

/**
* StopJob
* @param tag string
* @return error
**/
func StopJob(tag string) error {
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
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
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	return crontab.stop()
}

/**
* HttpGet
* @params w http.ResponseWriter, r *http.Request
**/
func HttpGet(w http.ResponseWriter, r *http.Request) {
	if getInstance == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "get instance not found")
		return
	}

	id := chi.URLParam(r, "id")
	var instance Job
	exists, err := getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}

/**
* HttpStart
* @params w http.ResponseWriter, r *http.Request
**/
func HttpStart(w http.ResponseWriter, r *http.Request) {
	if getInstance == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "get instance not found")
		return
	}

	id := chi.URLParam(r, "id")
	var instance Job
	exists, err := getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	err = StartJob(instance.Tag)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}

/**
* HttpStop
* @params w http.ResponseWriter, r *http.Request
**/
func HttpStop(w http.ResponseWriter, r *http.Request) {
	if getInstance == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "get instance not found")
		return
	}

	id := chi.URLParam(r, "id")
	var instance Job
	exists, err := getInstance(id, &instance)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": "instance not found"},
		})
		return
	}

	err = StopJob(instance.Tag)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: instance.ToJson(),
	})
}
