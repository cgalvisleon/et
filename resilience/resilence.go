package resilience

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type TpStore string

type Status string

const (
	packageName           = "resilience"
	TpStoreCache  TpStore = "cache"
	TpStoreMemory TpStore = "memory"
	StatusPending Status  = "pending"
	StatusRunning Status  = "running"
	StatusDone    Status  = "done"
	StatusStop    Status  = "stop"
	StatusFailed  Status  = "failed"
)

type Instance struct {
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	LastAttemptAt time.Time       `json:"last_attempt_at"`
	DoneAt        time.Time       `json:"done_at"`
	Id            string          `json:"id"`
	Tag           string          `json:"tag"`
	Description   string          `json:"description"`
	Status        Status          `json:"status"`
	TpStore       TpStore         `json:"store"`
	Attempt       int             `json:"attempt"`
	TotalAttempts int             `json:"total_attempts"`
	TimeAttempts  time.Duration   `json:"time_attempts"`
	RetentionTime time.Duration   `json:"retention_time"`
	Tags          et.Json         `json:"tags"`
	Team          string          `json:"team"`
	Level         string          `json:"level"`
	stop          bool            `json:"-"`
	err           error           `json:"-"`
	fn            interface{}     `json:"-"`
	fnArgs        []interface{}   `json:"-"`
	fnResult      []reflect.Value `json:"-"`
}

/**
* Instance
* @param id, tag, description string, totalAttempts int, timeAttempts, retentionTime time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}
* @return Instance
 */
func NewInstance(id, tag, description string, totalAttempts int, timeAttempts, retentionTime time.Duration, tags et.Json, team string, level string, fn interface{}, fnArgs ...interface{}) *Instance {
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
		RetentionTime: retentionTime,
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
	exists := cache.Exists(id)
	if !exists {
		return nil, fmt.Errorf(MSG_INSTANCE_NOT_FOUND)
	}

	bt, err := cache.Get(id, "")
	if err != nil {
		return nil, err
	}

	var result Instance
	err = json.Unmarshal([]byte(bt), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
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
		"id":              s.Id,
		"tag":             s.Tag,
		"description":     s.Description,
		"status":          s.Status,
		"tp_store":        s.TpStore,
		"attempt":         s.Attempt,
		"total_attempts":  s.TotalAttempts,
		"time_attempts":   s.TimeAttempts,
		"retention_time":  s.RetentionTime,
	}

	for k, v := range s.Tags {
		result[k] = v
	}

	return result
}

/**
* saveTo
* @param id string, bt []byte
**/
func (s *Instance) saveTo(id string, bt []byte) {
	if s.RetentionTime <= 0 {
		s.RetentionTime = 10 * time.Minute
	}

	err := cache.Set(id, string(bt), s.RetentionTime)
	if err != nil {
		mem.Set(id, string(bt), s.RetentionTime)
		s.TpStore = TpStoreMemory
	} else {
		s.TpStore = TpStoreCache
	}
}

/**
* save
* @return error
**/
func (s *Instance) save() error {
	event.Publish(EVENT_RESILIENCE_STATUS, s.ToJson())
	bt, err := json.Marshal(s)
	if err != nil {
		return err
	}

	s.saveTo(s.Id, bt)

	return nil
}

/**
* SetStatus
* @param status Status
* @return error
**/
func (s *Instance) setStatus(status Status) error {
	if s.Status == status {
		return nil
	}

	s.Status = status
	s.UpdatedAt = utility.NowTime()
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
			data := s.ToJson().Clone()
			data.Set("team", s.Team)
			data.Set("level", s.Level)
			message := fmt.Sprintf(MSG_RESILIENCE_FINISHED_ERROR, s.Attempt, s.TotalAttempts, s.Id, s.Tag, s.Status, errMsg)
			event.Publish(EVENT_RESILIENCE_FAILED, data)
			logs.Logf(packageName, message)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_ERROR, s.Attempt, s.TotalAttempts, s.Id, s.Tag, s.Status, errMsg)
		}
	default:
		if s.Attempt == s.TotalAttempts {
			logs.Logf(packageName, MSG_RESILIENCE_FINISHED, s.Attempt, s.TotalAttempts, s.Id, s.Tag, s.Status)
		} else {
			logs.Logf(packageName, MSG_RESILIENCE_STATUS, s.Attempt, s.TotalAttempts, s.Id, s.Tag, s.Status)
		}
	}

	return s.save()
}

/**
* setError
* @param err error
* @return error
**/
func (s *Instance) setError(err error) {
	s.err = err
	s.setStatus(StatusFailed)
}

/**
* setStop
* @return et.Item
**/
func (s *Instance) setStop() et.Item {
	s.stop = true
	s.setStatus(StatusStop)

	return et.Item{
		Ok:     true,
		Result: s.ToJson(),
	}
}

/**
* setRestart
* @return et.Item
**/
func (s *Instance) setRestart() et.Item {
	s.stop = false
	s.setStatus(StatusPending)
	go s.run()

	return et.Item{
		Ok:     true,
		Result: s.ToJson(),
	}
}

/**
* run
* @return []reflect.Value, error
**/
func (s *Instance) run() ([]reflect.Value, error) {
	if s.Status == StatusDone {
		return []reflect.Value{reflect.ValueOf(et.Item{
			Ok:     true,
			Result: s.ToJson(),
		})}, nil
	}

	if s.stop {
		return []reflect.Value{reflect.ValueOf(et.Item{
			Ok:     false,
			Result: s.ToJson(),
		})}, nil
	}

	s.LastAttemptAt = utility.NowTime()
	s.Attempt++
	s.setStatus(StatusRunning)

	argsValues := make([]reflect.Value, len(s.fnArgs))
	for i, arg := range s.fnArgs {
		argsValues[i] = reflect.ValueOf(arg)
	}

	var err error
	var ok bool
	fn := reflect.ValueOf(s.fn)
	s.fnResult = fn.Call(argsValues)
	for _, r := range s.fnResult {
		if r.Type().Implements(errorInterface) {
			err, ok = r.Interface().(error)
			if ok && err != nil {
				s.setError(err)
			}
		}
	}

	if s.Status != StatusFailed {
		s.done()
	}

	return s.fnResult, err
}

/**
* done
* @return error
**/
func (s *Instance) done() {
	s.setStatus(StatusDone)

	time.AfterFunc(3*time.Second, func() {
		delete(resilience, s.Id)
	})
}

/**
* runAttempt
* @return error
**/
func (s *Instance) runAttempt() {
	if s.TimeAttempts == 0 {
		return
	}

	time.AfterFunc(s.TimeAttempts, func() {
		if s.Status != StatusDone && s.Attempt < s.TotalAttempts {
			_, err := s.run()
			if err != nil {
				s.runAttempt()
			}
		}
	})
}

/**
* IsFailed
* @return bool
**/
func (s *Instance) IsFailed() bool {
	return s.Status == StatusFailed && s.Attempt == s.TotalAttempts
}

/**
* IsEnd
* @return bool
**/
func (s *Instance) IsEnd() bool {
	return s.Attempt == s.TotalAttempts
}
