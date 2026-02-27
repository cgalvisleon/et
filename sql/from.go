package sql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

type NextFn func(fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error

type Model struct {
	Host        string                                                                                            `json:"host"`
	Database    string                                                                                            `json:"database"`
	Schema      string                                                                                            `json:"schema"`
	Name        string                                                                                            `json:"name"`
	Hidden      []string                                                                                          `json:"hidden"`
	IdxField    string                                                                                            `json:"idx_field"`
	onNext      func(fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error `json:"-"`
	onGetIndex  func(field, key string, dest map[string]bool) (bool, error)                                       `json:"-"`
	onGetObject func(idx string, dest et.Json) (bool, error)                                                      `json:"-"`
}

/**
* Key
* @return string
**/
func (s *Model) Key() string {
	result := s.Name
	if s.Schema != "" {
		result = fmt.Sprintf("%s.%s", s.Schema, result)
	}
	if s.Database != "" {
		result = fmt.Sprintf("%s.%s", s.Database, result)
	}
	if s.Host != "" {
		result = fmt.Sprintf("%s:%s", s.Host, result)
	}
	return result
}

func (s *Model) OnNext(nextFn func(fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error) {
	s.onNext = nextFn
}

func (s *Model) OnGetIndex(getIndexFn func(field, key string, dest map[string]bool) (bool, error)) {
	s.onGetIndex = getIndexFn
}

func (s *Model) OnGetObject(getObjectFn func(idx string, dest et.Json) (bool, error)) {
	s.onGetObject = getObjectFn
}

func (s *Model) Next(fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error {
	return s.onNext(fn, asc, offset, limit, workers)
}

func (s *Model) GetIndex(field, key string, dest map[string]bool) (bool, error) {
	return s.onGetIndex(field, key, dest)
}

func (s *Model) GetObjet(idx string, dest et.Json) (bool, error) {
	return s.onGetObject(idx, dest)
}
