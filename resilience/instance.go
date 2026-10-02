package resilience

import (
	"fmt"
	"reflect"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type Status string

const (
	packageName        = "resilience"
	PENDING     Status = "pending"
	RUNNING     Status = "running"
	DONE        Status = "done"
	STOP        Status = "stop"
	FAILED      Status = "failed"
)

type Instance struct {
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ResilienceId  string          `json:"resilience_id"`
	ID            string          `json:"id"`
	Tag           string          `json:"tag"`
	Description   string          `json:"description"`
	LastAttemptAt time.Time       `json:"last_attempt_at"`
	DoneAt        time.Time       `json:"done_at"`
	Status        Status          `json:"status"`
	Attempt       int             `json:"attempt"`
	TotalAttempts int             `json:"total_attempts"`
	Interval      time.Duration   `json:"interval"`
	Tags          et.Json         `json:"tags"`
	Error         error           `json:"error"`
	Result        []any           `json:"result"`
	resilience    *Resilience     `json:"-"`
	stop          bool            `json:"-"`
	fn            interface{}     `json:"-"`
	fnArgs        []interface{}   `json:"-"`
	fnResult      []reflect.Value `json:"-"`
	isDebug       bool            `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
	errMsg := ""
	if s.Error != nil {
		errMsg = s.Error.Error()
	}
	result := et.Json{
		"created_at":      s.CreatedAt,
		"updated_at":      s.UpdatedAt,
		"resilience_id":   s.ResilienceId,
		"id":              s.ID,
		"tag":             s.Tag,
		"description":     s.Description,
		"last_attempt_at": s.LastAttemptAt,
		"done_at":         s.DoneAt,
		"status":          s.Status,
		"attempt":         s.Attempt,
		"total_attempts":  s.TotalAttempts,
		"interval":        s.Interval,
		"tags":            s.Tags,
		"error":           errMsg,
		"result":          s.Result,
	}
	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result
}

/**
* String
* @return string
**/
func (s *Instance) ToString() string {
	result := s.ToJson()
	return result.ToString()
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	if s.Status == status {
		return nil
	}

	s.Status = status
	s.UpdatedAt = timezone.Now()
	switch s.Status {
	case DONE:
		s.DoneAt = s.UpdatedAt
	case FAILED:
		logs.Logf(packageName, MSG_RESILIENCE_ERROR, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status, s.Error)
	default:
		if s.Attempt == s.TotalAttempts {
			logs.Logf(packageName, MSG_RESILIENCE_FINISHED, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_STATUS, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status)
		}
	}

	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	event.Publish(EVENT_INSTANCE_SET, data)
	bt, err := data.ToByte()
	if err != nil {
		return err
	}

	cache.Set(s.ID, bt, s.Interval)
	return nil
}

/**
* setError
* @param err error
**/
func (s *Instance) setError(err error) {
	s.Error = err
	s.setStatus(FAILED)
}

/**
* setDone
**/
func (s *Instance) setDone() {
	s.setStatus(DONE)
}

/**
* setStop
* @return et.Item
**/
func (s *Instance) setStop() {
	s.stop = true
	s.setStatus(STOP)
}

/**
* isStop
* @return bool
**/
func (s *Instance) isStop() bool {
	if s.stop {
		return true
	}

	key := fmt.Sprintf("resilience:%s:stop", s.ID)
	stop, _, err := cache.GetBool(key, false)
	if err != nil {
		return false
	}

	return stop
}

/**
* runAttempt
* @return []reflect.Value, error
**/
func (s *Instance) runAttempt() ([]any, error) {
	if s.Status == DONE {
		return s.Result, s.Error
	}

	if s.isStop() {
		return s.Result, s.Error
	}

	s.LastAttemptAt = timezone.Now()
	s.Attempt++
	s.setStatus(RUNNING)

	argsValues := make([]reflect.Value, len(s.fnArgs))
	for i, arg := range s.fnArgs {
		argsValues[i] = reflect.ValueOf(arg)
	}

	var err error
	var failed bool
	fn := reflect.ValueOf(s.fn)
	s.fnResult = fn.Call(argsValues)
	for _, r := range s.fnResult {
		if r.Type().Implements(errorInterface) {
			err, failed = r.Interface().(error)
		} else {
			s.Result = append(s.Result, r.Interface())
		}
	}

	if failed {
		s.setError(err)
	} else {
		s.setDone()
	}

	return s.Result, s.Error
}

/**
* Run
* @return error
**/
func (s *Instance) Run() ([]any, error) {
	defer func() {
		s.resilience.removeInstance(s.ID)
		cache.Delete(s.ID)
	}()

	if s.Interval == 0 {
		return s.Result, s.Error
	}

	time.AfterFunc(s.Interval, func() {
		if s.Status != DONE && s.Attempt < s.TotalAttempts {
			_, err := s.runAttempt()
			if err != nil {
				s.Run()
			}
		}
	})

	return s.Result, s.Error
}

/**
* isFailed
* @return bool
**/
func (s *Instance) IsFailed() bool {
	return s.Status == FAILED
}

/**
* IsDone
* @return bool
**/
func (s *Instance) IsDone() bool {
	result := s.Attempt == s.TotalAttempts
	if !result {
		result = s.Status == DONE
	}
	return result
}
