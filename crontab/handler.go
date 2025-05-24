package crontab

import (
	"errors"

	"github.com/cgalvisleon/et/et"
)

var crontab *Jobs

func Load() {
	if crontab != nil {
		return
	}

	crontab = New()
}

/**
* AddJob
* @param id, name, spec string, job func()
* @return error
**/
func AddJob(id, name, spec string, job func()) error {
	if crontab == nil {
		return errors.New("crontab not initialized")
	}

	return crontab.AddJob(id, name, spec, job)
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
