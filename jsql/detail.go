package jsql

type Detail struct {
	To              *From             `json:"to"`
	Keys            map[string]string `json:"keys"`
	Select          []interface{}     `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
	Page            int               `json:"page"`
	Rows            int               `json:"rows"`
}

/**
* setLimit
* @param page, rows int
* @return *Detail
**/
func (s *Detail) setLimit(page, rows int) *Detail {
	return &Detail{
		To:              s.To,
		Keys:            s.Keys,
		Select:          s.Select,
		OnDeleteCascade: s.OnDeleteCascade,
		OnUpdateCascade: s.OnUpdateCascade,
		Page:            page,
		Rows:            rows,
	}
}

/**
* newDetail
* @param to *Model, keys map[string]string, selecs []interface{}, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, keys map[string]string, selecs []interface{}, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		To:              getFrom(to, ""),
		Keys:            keys,
		Select:          selecs,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}

type TypeJoin string

const (
	JOIN  TypeJoin = "join"
	LEFT  TypeJoin = "left"
	RIGHT TypeJoin = "right"
	FULL  TypeJoin = "full"
)

type Joins struct {
	Type TypeJoin
	To   *From
	Keys map[string]string
}

/**
* newJoins
* @param tp TypeJoin, from *From, keys map[string]string
* @return *Joins
**/
func newJoins(tp TypeJoin, from *From, keys map[string]string) *Joins {
	return &Joins{
		Type: tp,
		To:   from,
		Keys: keys,
	}
}
