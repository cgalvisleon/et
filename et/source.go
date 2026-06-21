package et

type Source struct {
	data      []Json `json:"-"`
	as        string `json:"-"`
	pos       int    `json:"-"`
	limitRows int    `json:"-"`
}

/**
* Next
* @return (Json, bool)
**/
func (s *Source) Next() (Json, bool) {
	if s.pos >= len(s.data) {
		return Json{}, false
	}
	item := s.data[s.pos]
	s.pos++
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
* @param item Json
**/
func (s *Source) Add(item Json) {
	s.data = append(s.data, item)
}

/**
* Data
* @param index int
* @return Json
**/
func (s *Source) Data(index int) Json {
	if index < 0 || index >= len(s.data) {
		return Json{}
	}
	return s.data[index]
}

/**
* LimitRows
* @return int
**/
func (s *Source) LimitRows() int {
	if s.limitRows == 0 {
		s.limitRows = 1000
	}
	return s.limitRows
}
