package crontab

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

var crontab *Crontab

/**
* eventFunc
* @param job *Job, fn func(params et.Json) error
* @return error
**/
func eventFunc(job *Job, fn func(params et.Json) error) error {
	channel := fmt.Sprintf("job:%s:%s", job.TenantId, job.Tag)
	event.Subscribe(channel, func(msg event.Message) {
		params := msg.Data
		err := fn(params)
		if err != nil {
			logs.Log(packageName, fmt.Sprintf(MSG_ERROR_EXECUTING_JOB, job.Type, job.Tag, err))
		}
	})
	return nil
}

/**
* Load
* @params tag string, store Store
* @return error
**/
func Load(tenantId string, store Store) error {
	var err error
	crontab, err = New(tenantId, store)
	if err != nil {
		return err
	}

	return crontab.eventInit()
}

type Cron struct {
	DayOfWeek  string `json:"day_of_week"`
	Month      string `json:"mes"`
	DayOfMonth string `json:"day_of_month"`
	Hour       string `json:"hora"`
	Minute     string `json:"minuto"`
}

func (s *Cron) toString() (string, error) {
	dayOfWeekRegex := `^([0-7]|\*|\*/[0-7]|[0-7]-[0-7]|[0-7](,[0-7])*)$`
	monthRegex := `^([0-12]|\*|\*/[0-12]|[0-12]-[0-12]|[0-12](,[0-12])*)$`
	dayOfMonthRegex := `^([0-31]|\*|\*/[0-31]|[0-31]-[0-31]|[0-31](,[0-31])*)$`
	hourRegex := `^([0-23]|\*|\*/[0-23]|[0-23]-[0-23]|[0-23](,[0-23])*)$`
	minuteRegex := `^([0-59]|\*|\*/[0-59]|[0-59]-[0-59]|[0-59](,[0-59])*)$`

	if ok, _ := regexp.MatchString(dayOfWeekRegex, s.DayOfWeek); !ok {
		return "", errors.New(MSG_ERROR_DAY_OF_WEEK_INVALID)
	}

	if ok, _ := regexp.MatchString(monthRegex, s.Month); !ok {
		return "", errors.New(MSG_ERROR_MONTH_INVALID)
	}

	if ok, _ := regexp.MatchString(dayOfMonthRegex, s.DayOfMonth); !ok {
		return "", errors.New(MSG_ERROR_DAY_OF_MONTH_INVALID)
	}

	if ok, _ := regexp.MatchString(hourRegex, s.Hour); !ok {
		return "", errors.New(MSG_ERROR_HOUR_INVALID)
	}

	if ok, _ := regexp.MatchString(minuteRegex, s.Minute); !ok {
		return "", errors.New(MSG_ERROR_MINUTE_INVALID)
	}

	return fmt.Sprintf("%s %s %s %s %s", s.DayOfWeek, s.Month, s.DayOfMonth, s.Hour, s.Minute), nil
}

/**
* NewCronJob
* @param tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(params et.Json) error
* @return error
**/
func CronJob(tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(params et.Json) error) error {
	if crontab == nil {
		return errors.New(msg.MSG_CRONTAB_UNLOAD)
	}

	specStr, err := spec.toString()
	if err != nil {
		return err
	}

	job := newJob(crontab.TenantId, CRONJOB, tag, ownerId, specStr, params, repetitions)
	return eventFunc(job, fn)
}

/**
* AddCronJob
* @param tag, ownerId string, spec time.Time, repetitions int, params et.Json, fn func(params et.Json) error
* @return error
**/
func ScheduleJob(tag, ownerId string, spec time.Time, params et.Json, fn func(params et.Json) error) error {
	if crontab == nil {
		return errors.New(msg.MSG_CRONTAB_UNLOAD)
	}

	job := newJob(crontab.TenantId, SCHEDULEJOB, tag, ownerId, spec.Format(time.RFC3339), params, 0)
	return eventFunc(job, fn)
}

/**
* HttpSet
* @params w http.ResponseWriter, r *http.Request
**/
func HttpRemoveJob(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_CRONTAB_UNLOAD)
		return
	}

	id := request.URLParam(r, "id").Str()
	event.Publish(EVENT_CRONTAB_SET, et.Json{"id": id})

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_SEND_JOB_REMOVED},
	})
}

/**
* HttpStopJob
* @params w http.ResponseWriter, r *http.Request
**/
func HttpStopJob(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_CRONTAB_UNLOAD)
		return
	}

	id := request.URLParam(r, "id").Str()
	event.Publish(EVENT_CRONTAB_STOP, et.Json{"id": id})

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_SEND_JOB_STOPPED},
	})
}

/**
* HttpStartJob
* @params w http.ResponseWriter, r *http.Request
**/
func HttpStartJob(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_CRONTAB_UNLOAD)
		return
	}

	id := request.URLParam(r, "id").Str()
	event.Publish(EVENT_CRONTAB_START, et.Json{"id": id})

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_SEND_JOB_STARTED},
	})
}
