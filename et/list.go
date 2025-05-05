package et

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
	return s.ToJson().ToString()
}
