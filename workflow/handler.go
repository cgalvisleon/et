package workflow

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
)

var workFlows *WorkFlows

/**
* Load
* @return error
 */
func Load() error {
	if workFlows != nil {
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

	workFlows = newWorkFlows()
	return nil
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if err := Load(); err != nil {
		return false
	}

	return workFlows.healthCheck()
}

/**
* New
* @param tag, version, name, description string, fn FnContext, createdBy string
* @return *Flow
**/
func New(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	if err := Load(); err != nil {
		return nil
	}

	return workFlows.newFlow(tag, version, name, description, fn, stop, createdBy)
}

/**
* Run
* @param instanceId, tag string, startId int, tags et.Json, ctx et.Json, createdBy string
* @return et.Json, error
**/
func Run(instanceId, tag string, startId int, tags et.Json, ctx et.Json, createdBy string) (et.Json, error) {
	if err := Load(); err != nil {
		return et.Json{}, err
	}

	return workFlows.run(instanceId, tag, startId, tags, ctx, createdBy)
}

/**
* Continue
* @param instanceId string, tags et.Json, ctx et.Json, createdBy string
* @return et.Json, error
**/
func Continue(instanceId string, tags et.Json, ctx et.Json, createdBy string) (et.Json, error) {
	if err := Load(); err != nil {
		return et.Json{}, err
	}

	return workFlows.goOn(instanceId, tags, ctx, createdBy)
}

/**
* Rollback
* @param instanceId string
* @return et.Json, error
**/
func Rollback(instanceId string) (et.Json, error) {
	if err := Load(); err != nil {
		return et.Json{}, err
	}

	return workFlows.rollback(instanceId)
}

/**
* Stop
* @param instanceId, tag string
* @return error
**/
func Stop(instanceId string) error {
	if err := Load(); err != nil {
		return err
	}

	return workFlows.stop(instanceId)
}

/**
* DeleteFlow
* @param tag string
* @return (bool, error)
**/
func DeleteFlow(tag string) (bool, error) {
	if err := Load(); err != nil {
		return false, err
	}

	return workFlows.deleteFlow(tag), nil
}
