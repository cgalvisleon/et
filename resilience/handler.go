package resilience

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi"
)

var resilience map[string]*Instance

/**
* Load
* @return error
 */
func Load(store instances.Store) error {
	if resilience != nil {
		return nil
	}

	err := event.Load()
	if err != nil {
		return err
	}

	err = initEvents()
	if err != nil {
		return err
	}

	resilience = make(map[string]*Instance)
	if store != nil {
		SetGetInstance(store.Get)
		SetSetInstance(store.Set)
	}

	return nil
}

/**
* HealthCheck
* @return bool
 */
func HealthCheck() bool {
	if resilience == nil {
		return false
	}

	return true
}

/**
* AddCustom
* @param id, tag, description string, totalAttempts int, timeAttempts time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return *Instance
 */
func AddCustom(id, tag, description string, totalAttempts int, timeAttempts time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	if resilience == nil {
		return nil
	}

	result := NewInstance(id, tag, description, totalAttempts, timeAttempts, tags, team, level, fn, fnArgs...)
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

	return AddCustom(id, tag, description, totalAttempts, time.Duration(timeAttempts)*time.Second, tags, team, level, fn, fnArgs...)
}

/**
* Stop
* @param id string
* @return error
 */
func Stop(id string) error {
	if resilience == nil {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	if _, ok := resilience[id]; !ok {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
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
	if resilience == nil {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	if _, ok := resilience[id]; !ok {
		return fmt.Errorf(MSG_ID_NOT_FOUND)
	}

	resilience[id].setRestart()

	return nil
}

/**
* HttpGetResilienceById
* @param w http.ResponseWriter, r *http.Request
**/
func HttpGetResilienceById(w http.ResponseWriter, r *http.Request) {
	if resilience == nil {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_RESILIENCE_NOT_INITIALIZED)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_ID_REQUIRED)
		return
	}

	res, exist := LoadById(id)
	if !exist {
		response.HTTPError(w, r, http.StatusNotFound, MSG_ID_NOT_FOUND)
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
	if resilience == nil {
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
	if resilience == nil {
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
