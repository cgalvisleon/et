package et

type Source struct {
	data []Json
	as   string
}

/**
* Next
* @return (Json, bool)
**/
func (s *Source) Next() (Json, bool) {
	if len(s.data) == 0 {
		return Json{}, false
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
