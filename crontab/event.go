package crontab

import (
	"fmt"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	EVENT_CRONTAB_SERVER = "event:crontab:server"
	EVENT_CRONTAB_SET    = "event:crontab:set"
	EVENT_CRONTAB_STATUS = "event:crontab:status"
	EVENT_CRONTAB_STOP   = "event:crontab:stop"
	EVENT_CRONTAB_START  = "event:crontab:start"
	EVENT_CRONTAB_DELETE = "event:crontab:delete"
	EVENT_CRONTAB_FAILED = "event:crontab:failed"
)

/**
* eventInit
* @return error
**/
func eventInit() error {
	err := event.Stack(EVENT_CRONTAB_SET, eventSet)
	if err != nil {
		return err
	}

	err = event.Stack(EVENT_CRONTAB_DELETE, eventDelete)
	if err != nil {
		return err
	}

	err = event.Stack(EVENT_CRONTAB_STOP, eventStop)
	if err != nil {
		return err
	}

	err = event.Stack(EVENT_CRONTAB_START, eventStart)
	if err != nil {
		return err
	}

	return nil
}

/**
* eventSet
* @param msg event.Message
* @return error
**/
func eventSet(msg event.Message) {
	if crontab == nil {
		return
	}

	if !crontab.isServer {
		return
	}

	data := msg.Data
	id := data.Str("id")
	name := data.Str("name")
	spec := data.Str("spec")
	channel := data.Str("channel")
	started := data.Bool("started")
	params := data.Json("params")
	repetitions := data.Int("repetitions")
	_, err := crontab.addEventJob(id, name, spec, channel, started, params, repetitions)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Error adding job %s", err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s added", name))
}

/**
* eventDelete
* @param msg event.Message
* @return error
**/
func eventDelete(msg event.Message) {
	if crontab == nil {
		return
	}

	if !crontab.isServer {
		return
	}

	data := msg.Data
	id := data.Str("id")
	err := crontab.deleteJobById(id)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Error deleting job %s", err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s deleted", id))
}

/**
* eventStop
* @param msg event.Message
* @return error
**/
func eventStop(msg event.Message) {
	if crontab == nil {
		return
	}

	if !crontab.isServer {
		return
	}

	data := msg.Data
	id := data.Str("id")
	err := crontab.stopJobById(id)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Error stopping job %s", err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s stopped", id))
}

/**
* eventStart
* @param msg event.Message
* @return error
**/
func eventStart(msg event.Message) {
	if crontab == nil {
		return
	}

	if !crontab.isServer {
		return
	}

	data := msg.Data
	id := data.Str("id")
	err := crontab.startJobById(id)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Error starting job %s", err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s started", id))
}
