package et

type List struct {
	Rows   int    `json:"rows"`
	All    int    `json:"all"`
	Count  int    `json:"count"`
	Page   int    `json:"page"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Result []Json `json:"result"`
}

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
