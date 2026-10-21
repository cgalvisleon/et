package crontab

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/robfig/cron/v3"
)

var crontab *Crontab

// QUEUE_JOBS: la cola de job:<tag>; entre varias instancias, cada disparo lo atiende una sola
const QUEUE_JOBS = "crontab:jobs"

// specParser: el formato de los jobs cron (con segundos, como cron.WithSeconds) y los descriptores (@daily, @every 1h)
var specParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

/**
* channelOf: El canal en que se publica cada disparo de un job
* @param tag string
* @return string
**/
func channelOf(tag string) string {
	return fmt.Sprintf("job:%s", tag)
}

/**
* listen: Atiende los disparos de un job con fn, en la cola QUEUE_JOBS y con una sola suscripción por tag
* @param job *Job, fn func(params et.Json) error
* @return error
**/
func listen(job *Job, fn func(params et.Json) error) error {
	channel := channelOf(job.Tag)
	// event guarda una suscripción por canal: la anterior se cancela antes, o quedaría viva y repetiría cada disparo
	if err := event.Unsubscribe(channel); err != nil {
		return err
	}

	return event.Queue(channel, QUEUE_JOBS, func(msg event.Message) {
		params := msg.Data
		err := fn(params)
		if err != nil {
			logs.Log(packageName, fmt.Sprintf(MSG_ERROR_EXECUTING_JOB, job.Type, job.Tag, err))
		}
	})
}

/**
* register: Escucha los disparos del job y lo programa en esta instancia (la que lo registra, así quien lleva su estado
* es quien lo tiene); las demás quitan el registro anterior con el mismo tag (salvo esta revisión)
* @param job *Job, fn func(params et.Json) error
* @return error
**/
func register(job *Job, fn func(params et.Json) error) error {
	if err := listen(job, fn); err != nil {
		return err
	}

	replaced := et.Json{"id": job.ID, "keep": job.Rev}
	if err := event.Publish(EVENT_CRONTAB_REMOVE, replaced); err != nil {
		return err
	}

	return crontab.addJob(job)
}

/**
* Load
* @params tag string, store Store
* @return error
**/
func Load(tag string, store Store) error {
	var err error
	crontab, err = New(tag, store)
	if err != nil {
		return err
	}

	return nil
}

/**
* ParseSpec: Valida una expresión cron y la devuelve con segundos: 5 campos (minuto hora día mes día-semana) toman el
* segundo 0; también 6 campos y descriptores (@daily, @every 1h)
* @param spec string
* @return string, error
**/
func ParseSpec(spec string) (string, error) {
	fields := strings.Fields(spec)
	if len(fields) == 5 {
		fields = append([]string{"0"}, fields...)
	}

	result := strings.Join(fields, " ")
	if _, err := specParser.Parse(result); err != nil {
		return "", fmt.Errorf(MSG_ERROR_SPEC_INVALID, spec, err)
	}

	return result, nil
}

/**
* NextOf: La siguiente ejecución de una expresión cron después de from (en la zona de from)
* @param spec string, from time.Time
* @return time.Time, error
**/
func NextOf(spec string, from time.Time) (time.Time, error) {
	normal, err := ParseSpec(spec)
	if err != nil {
		return time.Time{}, err
	}

	schedule, err := specParser.Parse(normal)
	if err != nil {
		return time.Time{}, err
	}

	return schedule.Next(from), nil
}

/**
* CronJob: Programa un job repetitivo; el id del job es tag y registrarlo de nuevo lo reemplaza. En cada disparo fn
* recibe params (en una sola instancia); repetitions 0 es sin límite
* @param tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error
* @return *Job, error
**/
func CronJob(tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error) (*Job, error) {
	if crontab == nil {
		return nil, errors.New(msg.MSG_CRONTAB_UNLOAD)
	}

	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New(MSG_ERROR_TAG_REQUIRED)
	}

	spec, err := ParseSpec(spec)
	if err != nil {
		return nil, err
	}

	job := newJob(CRONJOB, tag, ownerId, spec, params, repetitions)
	return job, register(job, fn)
}

/**
* ScheduleJob: Programa un job de una sola vez en at (futura); el id del job es tag y registrarlo de nuevo lo reemplaza
* @param tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error
* @return *Job, error
**/
func ScheduleJob(tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error) (*Job, error) {
	if crontab == nil {
		return nil, errors.New(msg.MSG_CRONTAB_UNLOAD)
	}

	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New(MSG_ERROR_TAG_REQUIRED)
	}

	if !at.After(timezone.Now()) {
		return nil, fmt.Errorf(MSG_ERROR_SCHEDULE_PAST, at.Format(time.RFC3339))
	}

	job := newJob(SCHEDULEJOB, tag, ownerId, at.Format(time.RFC3339), params, 0)
	return job, register(job, fn)
}

/**
* RemoveJob: Quita el job de tag en todas las instancias y deja de atender sus disparos en esta
* @param tag string
* @return error
**/
func RemoveJob(tag string) error {
	if crontab == nil {
		return errors.New(msg.MSG_CRONTAB_UNLOAD)
	}

	channel := channelOf(tag)
	if err := event.Unsubscribe(channel); err != nil {
		return err
	}

	return event.Publish(EVENT_CRONTAB_REMOVE, et.Json{"id": tag})
}

/**
* HttpRemoveJob
* @params w http.ResponseWriter, r *http.Request
**/
func HttpRemoveJob(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_CRONTAB_UNLOAD)
		return
	}

	id := request.URLParam(r, "id").Str()
	event.Publish(EVENT_CRONTAB_REMOVE, et.Json{"id": id})

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
