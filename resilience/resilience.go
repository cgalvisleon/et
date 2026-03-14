package resilience

import (
	"fmt"
	"reflect"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

var packagePath = "resilience"

/**
* Instance
* @param id, tag, description string, totalAttempts int, timeAttempts time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return Instance
 */
func NewInstance(id, tag, description string, totalAttempts int, timeAttempts time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
	id = reg.GetUUID(id)
	result := &Instance{
		CreatedAt:     time.Now(),
		Id:            id,
		Tag:           tag,
		Description:   description,
		fn:            fn,
		fnArgs:        fnArgs,
		fnResult:      []reflect.Value{},
		TotalAttempts: totalAttempts,
		TimeAttempts:  timeAttempts,
		Tags:          tags,
		Team:          team,
		Level:         level,
		stop:          false,
	}
	result.setStatus(StatusPending)

	return result
}

/**
* LoadById
* @param id string
* @return *Instance, error
**/
func LoadById(id string) (*Instance, error) {
	if loadInstance == nil {
		return nil, fmt.Errorf("loadInstance function not set")
	}

	result, err := loadInstance(id)
	if err != nil {
		return nil, err
	}

	return result, nil
}
