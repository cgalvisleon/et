package jql

import "github.com/cgalvisleon/et/et"

type Source struct {
	data []et.Json
	as   string
}

/**
* Next
* @return (et.Json, bool)
**/
func (s *Source) Next() (et.Json, bool) {
	if len(s.data) == 0 {
		return et.Json{}, false
	}
	item := s.data[0]
	s.data = s.data[1:]
	return item, true
}

/**
* As
* @return string
**/
func (s *Source) As() string {
	return s.as
}

/**
* Add
* @param item et.Json
**/
func (s *Source) Add(item et.Json) {
	s.data = append(s.data, item)
}

/**
* Data
* @param index int
* @return et.Json
**/
func (s *Source) Data(index int) et.Json {
	if index < 0 || index >= len(s.data) {
		return et.Json{}
	}
	return s.data[index]
}
