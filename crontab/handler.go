package crontab

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/robfig/cron/v3"
)

// crontab: la instancia global de Load, detrás de las funciones del paquete.
var crontab *Crontab

// specParser: el formato de los jobs cron (con segundos, como cron.WithSeconds) y los descriptores (@daily, @every 1h)
var specParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

/**
* Load: Crea la instancia global (la que usan CronJob, ScheduleJob…); si ya había una, la cierra antes.
* @param tag string, store Store, opts ...Option
* @return error
**/
func Load(tag string, store Store, opts ...Option) error {
	if crontab != nil {
		crontab.Close()
		crontab = nil
	}

	result, err := New(tag, store, opts...)
	if err != nil {
		return err
	}
	crontab = result
	return nil
}

/**
* Default: La instancia global de Load, o nil si no se cargó.
* @return *Crontab
**/
func Default() *Crontab {
	return crontab
}

/**
* global: La instancia global o el error de que no está cargada.
* @return *Crontab, error
**/
func global() (*Crontab, error) {
	if crontab == nil {
		return nil, errors.New(msg.MSG_CRONTAB_UNLOAD)
	}
	return crontab, nil
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
* CronJob: Crontab.CronJob en la instancia global.
* @param tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error, opts ...JobOption
* @return *Job, error
**/
func CronJob(tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error, opts ...JobOption) (*Job, error) {
	c, err := global()
	if err != nil {
		return nil, err
	}
	return c.CronJob(tag, ownerId, spec, repetitions, params, fn, opts...)
}

/**
* ScheduleJob: Crontab.ScheduleJob en la instancia global.
* @param tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error, opts ...JobOption
* @return *Job, error
**/
func ScheduleJob(tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error, opts ...JobOption) (*Job, error) {
	c, err := global()
	if err != nil {
		return nil, err
	}
	return c.ScheduleJob(tag, ownerId, at, params, fn, opts...)
}

/**
* RemoveJob: Crontab.RemoveJob en la instancia global.
* @param tag string
* @return error
**/
func RemoveJob(tag string) error {
	c, err := global()
	if err != nil {
		return err
	}
	return c.RemoveJob(tag)
}

/**
* StopJob: Crontab.StopJob en la instancia global.
* @param tag string
* @return error
**/
func StopJob(tag string) error {
	c, err := global()
	if err != nil {
		return err
	}
	return c.StopJob(tag)
}

/**
* StartJob: Crontab.StartJob en la instancia global.
* @param tag string
* @return error
**/
func StartJob(tag string) error {
	c, err := global()
	if err != nil {
		return err
	}
	return c.StartJob(tag)
}

/**
* GetJob: Crontab.GetJob en la instancia global.
* @param tag string
* @return et.Json, bool
**/
func GetJob(tag string) (et.Json, bool) {
	c, err := global()
	if err != nil {
		return nil, false
	}
	return c.GetJob(tag)
}

/**
* HttpRemoveJob: Crontab.HttpRemoveJob en la instancia global.
* @param w http.ResponseWriter, r *http.Request
**/
func HttpRemoveJob(w http.ResponseWriter, r *http.Request) {
	httpGlobal(w, r, (*Crontab).HttpRemoveJob)
}

/**
* HttpStopJob: Crontab.HttpStopJob en la instancia global.
* @param w http.ResponseWriter, r *http.Request
**/
func HttpStopJob(w http.ResponseWriter, r *http.Request) {
	httpGlobal(w, r, (*Crontab).HttpStopJob)
}

/**
* HttpStartJob: Crontab.HttpStartJob en la instancia global.
* @param w http.ResponseWriter, r *http.Request
**/
func HttpStartJob(w http.ResponseWriter, r *http.Request) {
	httpGlobal(w, r, (*Crontab).HttpStartJob)
}

/**
* httpGlobal: Responde 400 si la instancia global no está cargada; si lo está, llama a handler.
* @param w http.ResponseWriter, r *http.Request, handler func(*Crontab, http.ResponseWriter, *http.Request)
**/
func httpGlobal(w http.ResponseWriter, r *http.Request, handler func(*Crontab, http.ResponseWriter, *http.Request)) {
	c, err := global()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	handler(c, w, r)
}
