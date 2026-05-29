package resilience

import (
	"fmt"
	"reflect"
	"sync"
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
	Tag           string          `json:"tag"`
	OwnerId       string          `json:"owner_id"`
	Description   string          `json:"description"`
	Status        Status          `json:"status"`
	Attempt       int             `json:"attempt"`
	TotalAttempts int             `json:"total_attempts"`
	Interval      time.Duration   `json:"interval"`
	Tags          et.Json         `json:"tags"`
	Team          string          `json:"team"`
	Level         string          `json:"level"`
	Error         string          `json:"error"`
	Response      map[string]any  `json:"response"`
	Result        any             `json:"result"`
	owner         *Resilience     `json:"-"`
	stop          bool            `json:"-"`
	err           error           `json:"-"`
	fn            interface{}     `json:"-"`
	fnArgs        []interface{}   `json:"-"`
	fnResult      []reflect.Value `json:"-"`
	isDebug       bool            `json:"-"`
	saveMu        sync.Mutex      `json:"-"`
	saveTimer     *time.Timer     `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Instance) ToJson() et.Json {
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
		"error":           s.Error,
		"response":        s.Response,
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
func (s *Instance) save() error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}

	s.saveTimer = time.AfterFunc(100*time.Millisecond, func() {
		data := s.ToJson()
		if s.isDebug {
			logs.Log(packageName, "save:", data.ToString())
		}

		if s.owner != nil && s.owner.store != nil {
			err := s.owner.store.Set(s.ID, s.Tag, s.OwnerId, data)
			if err != nil {
				logs.Errorf("Error saving instance resilience: %v", err)
			}
		}

		event.Publish(EVENT_INSTANCE_SET, data)
	})

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

	return s.save()
}

/**
* SetError
* @param err error
**/
func (s *Instance) SetError(err error) {
	s.Error = err.Error()
	s.err = err
	s.setStatus(FAILED)
}

/**
* Stop
* @return et.Item
**/
func (s *Instance) Stop() et.Item {
	s.stop = true
	s.setStatus(STOP)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_INSTANCE_STOPPED,
		},
	}
}

/**
* Restart
* @return et.Item
**/
func (s *Instance) Restart() et.Item {
	s.stop = false
	s.setStatus(PENDING)
	go s.Run()

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_INSTANCE_RESTARTED,
		},
	}
}

/**
* Done
* @return error
**/
func (s *Instance) Done() {
	if s.Response != nil && len(s.Response) > 0 {
		v := reflect.ValueOf(s.Response)
		s.Result = v.Interface()
	} else {
		s.Result = et.Json{}
	}

	s.setStatus(DONE)

	time.AfterFunc(300*time.Millisecond, func() {
		s.owner.remove(s.ID)
	})
}

/**
* runAttempt
* @return []reflect.Value, error
**/
func (s *Instance) runAttempt() ([]reflect.Value, error) {
	jsonData := s.ToJson()
	if s.Status == DONE {
		return []reflect.Value{reflect.ValueOf(et.Item{
			Ok:     true,
			Result: jsonData,
		})}, nil
	}

	if s.stop {
		return []reflect.Value{reflect.ValueOf(et.Item{
			Ok:     false,
			Result: jsonData,
		})}, nil
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
			if failed {
				s.SetError(err)
			} else {
				s.Done()
			}
		}
	}

	return s.fnResult, err
}

/**
* Run
* @return error
**/
func (s *Instance) Run() {
	if s.Interval == 0 {
		return
	}

	time.AfterFunc(s.Interval, func() {
		if s.Status != DONE && s.Attempt < s.TotalAttempts {
			_, err := s.runAttempt()
			if err != nil {
				s.Run()
			}
		}
	})
}

/**
* IsFailed
* @return bool
**/
func (s *Instance) IsFailed() bool {
	return s.Status == FAILED
}

/**
* IsEnd
* @return bool
**/
func (s *Instance) IsEnd() bool {
	result := s.Attempt == s.TotalAttempts
	if !result {
		result = s.Status == DONE
	}
	return result
}
