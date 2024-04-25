package et

// List struct to use in et
type List struct {
	Rows   int    `json:"rows"`
	All    int    `json:"all"`
	Count  int    `json:"count"`
	Page   int    `json:"page"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Result []Json `json:"result"`
}

// ToJson return the value of the key
func (it *List) ToJson() Json {
	return Json{
		"rows":   it.Rows,
		"all":    it.All,
		"count":  it.Count,
		"page":   it.Page,
		"start":  it.Start,
		"end":    it.End,
		"result": it.Result,
	}
}
