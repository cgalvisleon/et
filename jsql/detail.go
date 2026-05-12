package jsql

/**
* Detail: Defines a relationship to another model, including join keys and cascade rules.
**/
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
* setLimit: Returns a copy of the Detail with Page and Rows overridden.
* @param page int
* @param rows int
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
* newDetail: Constructs a Detail linking to the given model with join keys and cascade flags.
* @param to *Model
* @param keys map[string]string
* @param selects []interface{}
* @param onDeleteCascade bool
* @param onUpdateCascade bool
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

/**
* TypeJoin: Specifies the SQL join strategy.
**/
type TypeJoin string

const (
	JOIN  TypeJoin = "join"
	LEFT  TypeJoin = "left"
	RIGHT TypeJoin = "right"
	FULL  TypeJoin = "full"
)

/**
* Joins: Represents a single JOIN clause with its type, target table, and key mapping.
**/
type Joins struct {
	Type TypeJoin
	To   *From
	Keys map[string]string
}

/**
* newJoins: Constructs a Joins entry for the given type, From, and key mapping.
* @param tp TypeJoin
* @param from *From
* @param keys map[string]string
* @return *Joins
**/
func newJoins(tp TypeJoin, from *From, keys map[string]string) *Joins {
	return &Joins{
		Type: tp,
		To:   from,
		Keys: keys,
	}
}
