package et

import "encoding/json"

/* List struct to use in et */
type List struct {
	Rows   int    `json:"rows"`
	All    int    `json:"all"`
	Count  int    `json:"count"`
	Page   int    `json:"page"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Result []Json `json:"result"`
}

func (s *List) ToByte() ([]byte, error) {
	return json.Marshal(s)
}

/**
* ToJson
* @return Json
**/
func (s *List) ToJson() Json {
	bt, err := s.ToByte()
	if err != nil {
		return Json{}
	}

	var result Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return Json{}
	}
	return result
}

/**
* ToString
* @return string
**/
func (s *List) ToString() string {
	return s.ToJson().ToString()
}

/**
* ToMap
* @return map[string]interface{}
**/
func (s *List) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"rows":   s.Rows,
		"all":    s.All,
		"count":  s.Count,
		"page":   s.Page,
		"start":  s.Start,
		"end":    s.End,
		"result": s.Result,
	}
}

/**
* From
* @param as string
* @return *Where
**/
func (s List) From(as string) *Where {
	return From(s.Result, as)
}
