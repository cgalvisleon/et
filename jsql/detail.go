package jsql

import "github.com/cgalvisleon/et/et"

/**
* Detail: Defines a relationship to another model, including join keys and cascade rules.
**/
type Detail struct {
	To              *From             `json:"to"`
	Keys            map[string]string `json:"keys"`
	Select          []string          `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
	Rows            int               `json:"rows"`
}

/**
* GetQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *Detail) GetQuery(item et.Json, page, rows int) *Query {
	q := newQuery(s.To.Model, "A")
	for k, fk := range s.Keys {
		v, exists := item[k]
		if !exists {
			continue
		}
		q.Where(Eq(fk, v))
	}
	q.Select(s.Select...)
	q.Limit(rows)
	q.Page(page)
	return q
}

/**
* newDetail: Constructs a Detail linking to the given model with join keys and cascade flags.
* @param to *Model, keys map[string]string, selecs []string, onDeleteCascade bool, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool) *Detail {
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
