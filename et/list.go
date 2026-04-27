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

/**
* ToJson
* @return Json
**/
func (s *List) ToJson() Json {
	return Json{
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
* ToString
* @return string
**/
func (s *List) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(bt)
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
