package workflow

import (
	"os"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
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
* @param tag, version, name, description string, definition string, createdBy string
* @return *Flow
**/
func New(tag, version, name, description string, definition string, stop bool, createdBy string) *Flow {
	if err := Load(); err != nil {
		return nil
	}

	return workFlows.newFlowDefinition(tag, version, name, description, definition, stop, createdBy)
}

/**
* NewByFile
* @param tag, version, name, description string, filePath string, stop bool, createdBy string
* @return *Flow
**/
func NewByFile(tag, version, name, description string, filePath string, stop bool, createdBy string) *Flow {
	if err := Load(); err != nil {
		return nil
	}

	definition, err := os.ReadFile(filePath)
	if err != nil {
		logs.Error(err)
		definition = []byte("")
	}

	return workFlows.newFlowDefinition(tag, version, name, description, string(definition), stop, createdBy)
}

/**
* NewFn
* @param tag, version, name, description string, fn FnContext, createdBy string
* @return *Flow
**/
func NewFn(tag, version, name, description string, fn FnContext, stop bool, createdBy string) *Flow {
	if err := Load(); err != nil {
		return nil
	}

	return workFlows.newFlowFn(tag, version, name, description, fn, stop, createdBy)
}

/**
* DeleteFlow
* @param tag string
* @return error
**/
func DeleteFlow(tag string) error {
	if err := Load(); err != nil {
		return err
	}

	return workFlows.deleteFlow(tag)
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

	instance, err := workFlows.loadInstance(instanceId)
	if err != nil {
		return et.Json{}, err
	}

	return workFlows.run(instanceId, instance.Tag, instance.Current, tags, ctx, createdBy)
}

/**
* Reset
* @param instanceId string
* @return error
**/
func Reset(instanceId, createdBy string) error {
	if err := Load(); err != nil {
		return err
	}

	return workFlows.reset(instanceId, createdBy)
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
* @param instanceId string
* @return error
**/
func Stop(instanceId string) error {
	if err := Load(); err != nil {
		return err
	}

	return workFlows.stop(instanceId)
}

/**
* GetInstance
* @param instanceId string
* @return (*Instance, error)
**/
func GetInstance(instanceId string) (*Instance, error) {
	if err := Load(); err != nil {
		return nil, err
	}

	return workFlows.loadInstance(instanceId)
}

/**
* DeleteInstance
* @param instanceId string
* @return error
**/
func DeleteInstance(instanceId string) error {
	if err := Load(); err != nil {
		return err
	}

	return workFlows.delete(instanceId)
}
