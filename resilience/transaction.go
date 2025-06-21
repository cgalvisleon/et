package resilience

import (
	"reflect"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/utility"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type Store int

const (
	StoreCache Store = iota
	StoreMemory
)

func (s Store) String() string {
	return []string{"cache", "memory"}[s]
}

type TransactionStatus string

const (
	StatusPending TransactionStatus = "pending"
	StatusSuccess TransactionStatus = "success"
	StatusRunning TransactionStatus = "running"
	StatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	CreatedAt     time.Time         `json:"created_at"`
	LastAttemptAt time.Time         `json:"last_attempt_at"`
	Id            string            `json:"id"`
	Tag           string            `json:"tag"`
	Description   string            `json:"description"`
	Status        TransactionStatus `json:"status"`
	Store         Store             `json:"store"`
	Attempts      int               `json:"attempts"`
	fn            interface{}       `json:"-"`
	fnArgs        []interface{}     `json:"-"`
	fnResult      []reflect.Value   `json:"-"`
}

/**
* Json
* @return et.Json
**/
func (s *Transaction) Json() et.Json {
	return et.Json{
		"id":              s.Id,
		"tag":             s.Tag,
		"description":     s.Description,
		"status":          s.Status,
		"attempts":        s.Attempts,
		"store":           s.Store.String(),
		"created_at":      s.CreatedAt,
		"last_attempt_at": s.LastAttemptAt,
		"result":          s.fnResult,
	}
}

/**
* Transaction
* @param id, description string, fn interface{}, fnArgs ...interface{}
* @return Transaction
 */
func NewTransaction(tag, description string, fn interface{}, fnArgs ...interface{}) *Transaction {
	result := &Transaction{
		Id:            utility.UUID(),
		Tag:           tag,
		Description:   description,
		Status:        StatusPending,
		fn:            fn,
		fnArgs:        fnArgs,
		fnResult:      []reflect.Value{},
		CreatedAt:     time.Now(),
		LastAttemptAt: time.Now(),
	}

	result.save()
	return result
}

/**
* save
* @return error
**/
func (s *Transaction) save() error {
	err := cache.Set(s.Id, s.Json(), 0)
	if err != nil {
		mem.Set(s.Id, s.Json().ToString(), 0)
		s.Store = StoreMemory
	} else {
		s.Store = StoreCache
	}

	return nil
}

/**
* done
* @return error
**/
func (s *Transaction) Done() error {
	if s.Store == StoreCache {
		_, err := cache.Delete(s.Id)
		if err != nil {
			return err
		}
	} else {
		mem.Delete(s.Id)
	}

	return nil
}

/**
* SetStatus
* @param status TransactionStatus
* @return error
**/
func (s *Transaction) setStatus(status TransactionStatus) error {
	s.Status = status
	return s.save()
}

/**
* Run
* @return error
**/
func (s *Transaction) Run() ([]reflect.Value, error) {
	if s.Status == StatusSuccess {
		return []reflect.Value{reflect.ValueOf(et.Item{
			Ok:     true,
			Result: s.Json(),
		})}, nil
	}

	s.LastAttemptAt = utility.NowTime()
	s.Attempts++
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
				s.setStatus(StatusFailed)
			}
		}
	}

	if s.Status != StatusFailed {
		s.setStatus(StatusSuccess)
	}

	logs.Log("resilience", "run:", s.Json().ToString())
	return s.fnResult, err
}
