package resilience

import (
	"fmt"
	"reflect"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
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
	LastAttemptAt time.Time       `json:"last_attempt_at"`
	DoneAt        time.Time       `json:"done_at"`
	ID            string          `json:"id"`
	OwnerId       string          `json:"owner_id"`
	TenantId      string          `json:"tenant_id"`
	Tag           string          `json:"tag"`
	Description   string          `json:"description"`
	Status        Status          `json:"status"`
	Attempt       int             `json:"attempt"`
	TotalAttempts int             `json:"total_attempts"`
	Interval      time.Duration   `json:"interval"`
	Tags          et.Json         `json:"tags"`
	Team          string          `json:"team"`
	Level         string          `json:"level"`
	Error         error           `json:"error"`
	Result        []any           `json:"result"`
	owner         *Resilience     `json:"-"`
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
		"last_attempt_at": s.LastAttemptAt,
		"done_at":         s.DoneAt,
		"id":              s.ID,
		"tag":             s.Tag,
		"owner_id":        s.OwnerId,
		"description":     s.Description,
		"status":          s.Status,
		"attempt":         s.Attempt,
		"total_attempts":  s.TotalAttempts,
		"interval":        s.Interval,
		"tags":            s.Tags,
		"team":            s.Team,
		"level":           s.Level,
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
* save
* @return error
**/
func (s *Instance) save(userId string) error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.store != nil {
		err := s.owner.store.Set(s.ID, s.Tag, s.TenantId, s.OwnerId, data, userId)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_INSTANCE_SET, data)

	return nil
}

/**
* up
* @param owner *Resilience
* @return *Instance
**/
func (s *Instance) up(owner *Resilience) *Instance {
	s.owner = owner
	return s
}

/**
* setStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status, userId string) error {
	if s.Status == status {
		return nil
	}

	s.Status = status
	s.UpdatedAt = timezone.Now()
	switch s.Status {
	case DONE:
		s.DoneAt = s.UpdatedAt
	case FAILED:
		if s.Attempt == s.TotalAttempts {
			data := s.ToJson()
			data.Set("team", s.Team)
			data.Set("level", s.Level)
			message := fmt.Sprintf(MSG_RESILIENCE_FINISHED_ERROR, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status, s.Error)
			event.Publish(EVENT_RESILIENCE_FAILED, data)
			logs.Logf(packageName, message)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_ERROR, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status, s.Error)
		}
	default:
		if s.Attempt == s.TotalAttempts {
			logs.Logf(packageName, MSG_RESILIENCE_FINISHED, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_STATUS, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status)
		}
	}

	return s.save(userId)
}

/**
* setError
* @param err error
**/
func (s *Instance) setError(err error, userId string) {
	s.Error = err
	s.setStatus(FAILED, userId)
}

/**
* setDone
**/
func (s *Instance) setDone(userId string) {
	s.setStatus(DONE, userId)

	time.AfterFunc(300*time.Millisecond, func() {
		s.owner.removeInstance(s.ID)
	})
}

/**
* setStop
* @return et.Item
**/
func (s *Instance) setStop(userId string) et.Item {
	s.stop = true
	s.setStatus(STOP, userId)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_INSTANCE_STOPPED,
		},
	}
}

/**
* setRestart
* @return et.Item
**/
func (s *Instance) setRestart(userId string) ([]any, error) {
	s.stop = false
	s.setStatus(PENDING, userId)
	return s.Run(userId)
}

/**
* runAttempt
* @return []reflect.Value, error
**/
func (s *Instance) runAttempt(userId string) ([]any, error) {
	if s.Status == DONE {
		return s.Result, s.Error
	}

	if s.stop {
		return s.Result, s.Error
	}

	s.LastAttemptAt = timezone.Now()
	s.Attempt++
	s.setStatus(RUNNING, userId)

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
		s.setError(err, userId)
	} else {
		s.setDone(userId)
	}

	return s.Result, err
}

/**
* Run
* @return error
**/
func (s *Instance) Run(userId string) ([]any, error) {
	if s.Interval == 0 {
		return s.Result, s.Error
	}

	time.AfterFunc(s.Interval, func() {
		if s.Status != DONE && s.Attempt < s.TotalAttempts {
			_, err := s.runAttempt(userId)
			if err != nil {
				s.Run(userId)
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
