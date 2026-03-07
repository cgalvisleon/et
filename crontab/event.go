package crontab

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	EVENT_CRONTAB_SET    = "event:crontab:set"
	EVENT_CRONTAB_REMOVE = "event:crontab:remove"
	EVENT_CRONTAB_STOP   = "event:crontab:stop"
	EVENT_CRONTAB_START  = "event:crontab:start"
)

/**
* eventInit
* @return error
**/
func (s *Jobs) eventInit() error {
	EVENT_CRONTAB_SET = fmt.Sprintf("event:crontab:set:%s", s.Tag)
	EVENT_CRONTAB_REMOVE = fmt.Sprintf("event:crontab:remove:%s", s.Tag)
	EVENT_CRONTAB_STOP = fmt.Sprintf("event:crontab:stop:%s", s.Tag)
	EVENT_CRONTAB_START = fmt.Sprintf("event:crontab:start:%s", s.Tag)

	err := event.Stack(EVENT_CRONTAB_SET, s.eventSet)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_REMOVE, s.eventRemove)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_STOP, s.eventStop)
	if err != nil {
		return err
	}

	err = event.Subscribe(EVENT_CRONTAB_START, s.eventStart)
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
func (s *Jobs) eventSet(msg event.Message) {
	n := cache.Incr(msg.Channel, 3*time.Minute)
	if n != 1 {
		logs.Errorf(packageName, "eventSet: %s", "job already exists")
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

	err := RemoveJob(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("%s: %s; Error removing job %s", tpStr, tag, err))
		return
	}

	_, err = s.addJob(tp, tag, spec, channel, started, params, repetitions)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("%s: %s; Error adding job %s", tpStr, tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("%s: %s added spec %s", tpStr, tag, spec))
}

/**
* eventRemove
* @param msg event.Message
* @return error
**/
func (s *Jobs) eventRemove(msg event.Message) {
	data := msg.Data
	tag := data.Str("tag")
	err := s.removeJob(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error removing job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s removed", tag))
}

/**
* eventStop
* @param msg event.Message
* @return error
**/
func (s *Jobs) eventStop(msg event.Message) {
	data := msg.Data
	tag := data.Str("tag")
	err := s.stopJob(tag)
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
func (s *Jobs) eventStart(msg event.Message) {
	data := msg.Data
	tag := data.Str("tag")
	err := s.startJob(tag)
	if err != nil {
		logs.Logf(packageName, fmt.Sprintf("Crontab %s; Error starting job %s", tag, err))
		return
	}

	logs.Logf(packageName, fmt.Sprintf("Crontab %s started", tag))
}
