package crontab

import (
	"fmt"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	EVENT_CRONTAB_SET    = "event:crontab:set"
	EVENT_CRONTAB_DELETE = "event:crontab:delete"
	EVENT_CRONTAB_STOP   = "event:crontab:stop"
	EVENT_CRONTAB_START  = "event:crontab:start"
)

/**
* eventInit
* @return error
**/
func eventInit(tag string) error {
	EVENT_CRONTAB_SET = fmt.Sprintf("event:crontab:set:%s", tag)
	EVENT_CRONTAB_DELETE = fmt.Sprintf("event:crontab:delete:%s", tag)
	EVENT_CRONTAB_STOP = fmt.Sprintf("event:crontab:stop:%s", tag)
	EVENT_CRONTAB_START = fmt.Sprintf("event:crontab:start:%s", tag)

	err := event.Stack(EVENT_CRONTAB_SET, eventSet)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_DELETE, eventDelete)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_STOP, eventStop)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_START, eventStart)
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

	data := msg.Data
	tpStr := data.Str("type")
	tag := data.Str("tag")
	spec := data.Str("spec")
	channel := data.Str("channel")
	started := data.Bool("started")
	params := data.Json("params")
	repetitions := data.Int("repetitions")
	tp := TypeJob(tpStr)
	_, err := crontab.addEventJob(tp, tag, spec, channel, started, params, repetitions, nil)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error adding job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s added", tag))
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

	data := msg.Data
	tag := data.Str("tag")
	err := crontab.deleteJobByTag(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error deleting job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s deleted", tag))
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

	data := msg.Data
	tag := data.Str("tag")
	err := crontab.stopJobByTag(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error stopping job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s stopped", tag))
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

	data := msg.Data
	tag := data.Str("tag")
	_, err := crontab.startJobByTag(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error starting job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s started", tag))
}
