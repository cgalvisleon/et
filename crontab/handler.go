package crontab

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
)

var crontab *Jobs

/**
* Load
**/
func Load() {
	crontab = New()
	crontab.Load()
}

/**
* Save
**/
func Save() {
	if crontab == nil {
		return
	}

	crontab.Save()
}

/**
* AddJob
* @param name, spec, channel string, params et.Json
* @return *Job, error
**/
func AddJob(name, spec, channel string, params et.Json) (*Job, error) {
	if crontab == nil {
		return nil, errors.New("crontab not initialized")
	}

	return crontab.AddJob(name, spec, channel, params, nil)
}

/**
* AddFnJob
* @param name, spec, channel string, params et.Json
* @return *Job, error
**/
func AddFnJob(name, spec, channel string, params et.Json, fn func()) (*Job, error) {
	if crontab == nil {
		return nil, errors.New("crontab not initialized")
	}

	return crontab.AddJob(name, spec, channel, params, fn)
}

/**
* DeleteJob
* @param name string
* @return error
**/
func DeleteJob(name string) error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.DeleteJob(name)
}

/**
* DeleteJobById
* @param id string
* @return error
**/
func DeleteJobById(id string) error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.DeleteJobById(id)
}

/**
* StartJob
* @param name string
* @return int, error
**/
func StartJob(name string) (int, error) {
	if crontab == nil {
		return 0, errors.New("crontab not initialized")
	}

	return crontab.StartJob(name)
}

/**
* StartJobById
* @param id string
* @return int, error
**/
func StartJobById(id string) (int, error) {
	if crontab == nil {
		return 0, errors.New("crontab not initialized")
	}

	return crontab.StartJobById(id)
}

/**
* StopJob
* @param name string
* @return error
**/
func StopJob(name string) error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.StopJob(name)
}

/**
* StopJobById
* @param id string
* @return error
**/
func StopJobById(id string) error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.StopJobById(id)
}

/**
* ListJobs
* @return et.Items, error
**/
func ListJobs() (et.Items, error) {
	if crontab == nil {
		return et.Items{}, errors.New("crontab not initialized")
	}

	return crontab.List(), nil
}

/**
* Start
* @return error
**/
func Start() error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.Start()
}

/**
* Stop
* @return error
**/
func Stop() error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.Stop()
}

/**
* HttpCrontabs
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpCrontabs(w http.ResponseWriter, r *http.Request) {
	if crontab == nil {
		response.JSON(w, r, http.StatusInternalServerError, et.Json{
			"message": "crontab not initialized",
		})
		return
	}

	result := et.Items{}
	for _, job := range crontab.jobs {
		result.Add(job.Json())
	}

	response.ITEMS(w, r, http.StatusOK, result)
}
