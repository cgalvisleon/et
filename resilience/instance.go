package resilience

import (
	"encoding/json"
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
	packageName          = "resilience"
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusStop    Status = "stop"
	StatusFailed  Status = "failed"
)

type Instance struct {
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	LastAttemptAt time.Time       `json:"last_attempt_at"`
	DoneAt        time.Time       `json:"done_at"`
	ID            string          `json:"id"`
	Tag           string          `json:"tag"`
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
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Instance) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	for k, v := range s.Tags {
		result.Set(k, v)
	}

	return result, nil
}

/**
* String
* @return string
**/
func (s *Instance) ToString() string {
	result, err := s.ToJson()
	if err != nil {
		return ""
	}

	return result.ToString()
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	data, err := s.ToJson()
	if err != nil {
		return err
	}

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.store != nil {
		return s.owner.store.Set(s.ID, s.Tag, s)
	}

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
	if s.Status == StatusDone {
		s.DoneAt = s.UpdatedAt
	}

	switch s.Status {
	case StatusFailed:
		errMsg := ""
		if s.err != nil {
			errMsg = s.err.Error()
		}
		if s.Attempt == s.TotalAttempts {
			data, err := s.ToJson()
			if err != nil {
				return err
			}
			data.Set("team", s.Team)
			data.Set("level", s.Level)
			message := fmt.Sprintf(MSG_RESILIENCE_FINISHED_ERROR, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status, errMsg)
			event.Publish(EVENT_RESILIENCE_FAILED, data)
			logs.Logf(packageName, message)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_ERROR, s.Attempt, s.TotalAttempts, s.ID, s.Tag, s.Status, errMsg)
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
	s.setStatus(StatusFailed)
}

/**
* Stop
* @return et.Item
**/
func (s *Instance) Stop() et.Item {
	s.stop = true
	s.setStatus(StatusStop)

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
	s.setStatus(StatusPending)
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

	s.setStatus(StatusDone)

	time.AfterFunc(3*time.Second, func() {
		s.owner.remove(s.ID)
	})
}

/**
* runAttempt
* @return []reflect.Value, error
**/
func (s *Instance) runAttempt() ([]reflect.Value, error) {
	jsonData, err := s.ToJson()
	if err != nil {
		return nil, err
	}

	if s.Status == StatusDone {
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
	s.setStatus(StatusRunning)

	argsValues := make([]reflect.Value, len(s.fnArgs))
	for i, arg := range s.fnArgs {
		argsValues[i] = reflect.ValueOf(arg)
	}

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
		if s.Status != StatusDone && s.Attempt < s.TotalAttempts {
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
	return s.Status == StatusFailed
}

/**
* IsEnd
* @return bool
**/
func (s *Instance) IsEnd() bool {
	result := s.Attempt == s.TotalAttempts
	if !result {
		result = s.Status == StatusDone
	}
	return result
}
