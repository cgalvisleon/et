package crontab

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

var crontab *Crontab

/**
* Load
* @params db *jdb.DB, schemaName, tag string
* @return error
**/
func Load(tag string, store instances.Store) error {
	var err error
	crontab, err = New(tag)
	if err != nil {
		return err
	}

	err = crontab.start()
	if err != nil {
		return err
	}

	if store != nil {
		SetGetInstance(store.Get)
		SetSetInstance(store.Set)
	}

	time.Sleep(1 * time.Second)

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

	logs.Log(packageName, `Disconnect...`)
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
* AddEventJob
* @param tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)
* @return error
**/
func AddEventJob(tag, spec string, repetitions int, started bool, params et.Json, fn func(event.Message)) error {
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	channel := fmt.Sprintf("cronjob:%s", tag)
	return crontab.addEventJob(CronJob, tag, spec, channel, started, params, repetitions, fn)
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
	if crontab == nil {
		return fmt.Errorf(msg.MSG_CRONTAB_UNLOAD)
	}

	channel := fmt.Sprintf("schedule:%s", tag)
	return crontab.addEventJob(ScheduleJob, tag, schedule, channel, started, params, 0, fn)
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
* HttpSet
* @params w http.ResponseWriter, r *http.Request
**/
func HttpSet(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_CRONTAB_UNLOAD)
		return
	}

	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	typeJob := body.Str("type")
	tag := body.Str("tag")
	spec := body.Str("spec")
	started := body.Bool("started")
	params := body.Json("params")
	repetitions := body.Int("repetitions")
	channel := fmt.Sprintf("cronjob:%s", tag)
	err = crontab.addEventJob(TypeJob(typeJob), tag, spec, channel, started, params, repetitions, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	id := request.URLParam(r, "id").Str()
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
* HttpGet
* @params w http.ResponseWriter, r *http.Request
**/
func HttpGet(w http.ResponseWriter, r *http.Request) {
	if getInstance == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "get instance not found")
		return
	}

	id := request.URLParam(r, "id").Str()
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

	id := request.URLParam(r, "id").Str()
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

	id := request.URLParam(r, "id").Str()
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
