package resilience

import (
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
* @return *Instance, bool
**/
func LoadById(id string) (*Instance, bool) {
	if id == "" {
		return nil, false
	}

	result, ok := resilience[id]
	if ok {
		return result, true
	}

	if getInstance != nil {
		var result Instance
		ok, err := getInstance(id, &result)
		if err != nil {
			return nil, false
		}

		if !ok {
			return nil, false
		}

		return &result, true
	}

	return nil, false
}
