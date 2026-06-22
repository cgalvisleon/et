package crontab

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

var (
	EVENT_CRONTAB_SET    = "crontab:set"
	EVENT_CRONTAB_REMOVE = "crontab:remove"
	EVENT_CRONTAB_STOP   = "crontab:stop"
	EVENT_CRONTAB_START  = "crontab:start"
	EVENT_INSTANCE_SET   = "crontab:instance:set"
)

/**
* eventInit
* @return error
**/
func (s *Crontab) eventInit() error {
	EVENT_CRONTAB_SET = fmt.Sprintf("crontab:set:%s", s.Tag)
	EVENT_CRONTAB_REMOVE = fmt.Sprintf("crontab:remove:%s", s.Tag)
	EVENT_CRONTAB_STOP = fmt.Sprintf("crontab:stop:%s", s.Tag)
	EVENT_CRONTAB_START = fmt.Sprintf("crontab:start:%s", s.Tag)

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
func (s *Crontab) eventSet(msg event.Message) {
	data := msg.Data
	bt := []byte(data.ToString())
	var job *Job
	err := json.Unmarshal(bt, &job)
	if err != nil {
		logs.Errorf(MSG_ERROR_UNMARSHALLING_JOB, err.Error())
		return
	}

	err = s.addJob(job)
	if err != nil {
		logs.Errorf(MSG_ERROR_ADDING_JOB, err.Error())
		return
	}
}

/**
* eventRemove
* @param msg event.Message
* @return error
**/
func (s *Crontab) eventRemove(msg event.Message) {
	data := msg.Data
	id := data.Str("id")
	s.removeJob(id)
}

/**
* eventStop
* @param msg event.Message
* @return error
**/
func (s *Crontab) eventStop(msg event.Message) {
	data := msg.Data
	id := data.Str("id")
	err := s.stopJob(id)
	if err != nil {
		logs.Errorf(MSG_ERROR_STOPPING_JOB, err.Error())
		return
	}
}

/**
* eventStart
* @param msg event.Message
* @return error
**/
func (s *Crontab) eventStart(msg event.Message) {
	data := msg.Data
	id := data.Str("id")
	err := s.startJob(id)
	if err != nil {
		logs.Errorf(MSG_ERROR_STARTING_JOB, err.Error())
		return
	}
}
