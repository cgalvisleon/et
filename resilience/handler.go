package resilience

import (
	"errors"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

var resilience map[string]*Instance

/**
* load
* @return error
 */
func load() error {
	if resilience != nil {
		return nil
	}

	err := cache.Load()
	if err != nil {
		return err
	}

	err = event.Load()
	if err != nil {
		return err
	}

	initEvents()

	resilience = make(map[string]*Instance)

	return nil
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if err := load(); err != nil {
		return false
	}

	if !cache.HealthCheck() {
		return false
	}

	if !event.HealthCheck() {
		return false
	}

	return true
}

/**
* AddCustom
* @param id, tag, description string, totalAttempts int, timeAttempts, retentionTime time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func AddCustom(id, tag, description string, totalAttempts int, timeAttempts, retentionTime time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if err := load(); err != nil {
		return nil
	}

	result := NewInstance(id, tag, description, totalAttempts, timeAttempts, retentionTime, tags, team, level, fn, fnArgs...)
	resilience[id] = result
	result.runAttempt()

	return result
}

/**
* Add
* @param tag, description string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func Add(id, tag, description string, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	totalAttempts := envar.GetInt("RESILIENCE_TOTAL_ATTEMPTS", 3)
	timeAttempts := envar.GetInt("RESILIENCE_TIME_ATTEMPTS", 30)
	retentionTime := envar.GetInt("RESILIENCE_RETENTION_TIME", 10)

	return AddCustom(id, tag, description, totalAttempts, time.Duration(timeAttempts)*time.Second, time.Duration(retentionTime)*time.Minute, tags, team, level, fn, fnArgs...)
}

/**
* Stop
* @param id string
* @return error
 */
func Stop(id string) error {
	if err := load(); err != nil {
		return err
	}

	if _, ok := resilience[id]; !ok {
		return errors.New(MSG_ID_NOT_FOUND)
	}

	resilience[id].setStop()

	return nil
}

/**
* Restart
* @param id string
* @return error
 */
func Restart(id string) error {
	if err := load(); err != nil {
		return err
	}

	if _, ok := resilience[id]; !ok {
		return errors.New(MSG_ID_NOT_FOUND)
	}

	resilience[id].setRestart()

	return nil
}

/**
* HttpGetResilienceById
* @param w http.ResponseWriter, r *http.Request
**/
func HttpGetResilienceById(w http.ResponseWriter, r *http.Request) {
	if err := load(); err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_RESILIENCE_NOT_INITIALIZED)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_ID_REQUIRED)
		return
	}

	res, err := LoadById(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusNotFound, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: res.ToJson(),
	})
}

/**
* HttpGetResilienceStop
* @param w http.ResponseWriter, r *http.Request
**/
func HttpGetResilienceStop(w http.ResponseWriter, r *http.Request) {
	if err := load(); err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_RESILIENCE_NOT_INITIALIZED)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_ID_REQUIRED)
		return
	}

	res, ok := resilience[id]
	if !ok {
		response.HTTPError(w, r, http.StatusNotFound, MSG_ID_NOT_FOUND)
		return
	}

	result := res.setStop()
	response.ITEM(w, r, http.StatusOK, result)
}

/**
* HttpGetResilienceRestart
* @param w http.ResponseWriter, r *http.Request
**/
func HttpGetResilienceRestart(w http.ResponseWriter, r *http.Request) {
	if err := load(); err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_RESILIENCE_NOT_INITIALIZED)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_ID_REQUIRED)
		return
	}

	res, ok := resilience[id]
	if !ok {
		response.HTTPError(w, r, http.StatusNotFound, MSG_ID_NOT_FOUND)
		return
	}

	result := res.setRestart()
	response.ITEM(w, r, http.StatusOK, result)
}
