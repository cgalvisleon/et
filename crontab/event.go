package crontab

import (
	"fmt"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	EVENT_CRONTAB_SET    = "event:crontab:set"
	EVENT_CRONTAB_REMOVE = "event:crontab:remove"
	EVENT_CRONTAB_STOP   = "event:crontab:stop"
	EVENT_CRONTAB_START  = "event:crontab:start"
	EVENT_CRONTAB_STATUS = "event:crontab:status"
)

/**
* eventInit
* @return error
**/
func eventInit(tag string) error {
	EVENT_CRONTAB_SET = fmt.Sprintf("event:crontab:set:%s", tag)
	EVENT_CRONTAB_REMOVE = fmt.Sprintf("event:crontab:remove:%s", tag)
	EVENT_CRONTAB_STOP = fmt.Sprintf("event:crontab:stop:%s", tag)
	EVENT_CRONTAB_START = fmt.Sprintf("event:crontab:start:%s", tag)

	err := event.Stack(EVENT_CRONTAB_SET, eventSet)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_REMOVE, eventRemove)
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
	params := data.Json("params")
	repetitions := data.Int("repetitions")
	tp := TypeJob(tpStr)
	_, err := crontab.addJob(tp, tag, spec, channel, params, repetitions)
	if err != nil {
		logs.Logf(packageName, "Crontab %s; Error adding job %s", tag, err)
		return
	}

	logs.Logf(packageName, "Crontab %s added", tag)
}

/**
* eventRemove
* @param msg event.Message
* @return error
**/
func eventRemove(msg event.Message) {
	if crontab == nil {
		return
	}

	data := msg.Data
	tag := data.Str("tag")
	err := crontab.removeJob(tag)
	if err != nil {
		logs.Logf(packageName, "Crontab %s; Error deleting job %s", tag, err)
		return
	}

	logs.Logf(packageName, "Crontab %s deleted", tag)
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
	err := crontab.stopJob(tag)
	if err != nil {
		logs.Logf(packageName, "Crontab %s; Error stopping job %s", tag, err)
		return
	}

	logs.Logf(packageName, "Crontab %s stopped", tag)
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
	err := crontab.startJob(tag)
	if err != nil {
		logs.Logf(packageName, "Crontab %s; Error starting job %s", tag, err)
		return
	}

	logs.Logf(packageName, "Crontab %s started", tag)
}
