package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/et"
)

func DetailKeys(key, foreignKey string) map[string]string {
	return map[string]string{
		key: foreignKey,
	}
}

func RollupKeys(key, foreignKey string) map[string]string {
	return map[string]string{
		foreignKey: key,
	}
}

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
* Ref: Returns the reference of the detail.
* @return et.Json
**/
func (s *Detail) Ref() et.Json {
	return et.Json{
		"to": s.To,
	}
}

/**
* init: Initializes the detail.
* @return error
**/
func (s *Detail) init() error {
	if s.To == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if s.To.Model == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	err := s.To.Model.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* GetQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *Detail) GetQuery(item et.Json, page, rows int) *Query {
	q := NewQuery(s.To.Model, "A")
	for k, fk := range s.Keys {
		v, exists := item[k]
		if !exists {
			continue
		}
		q.Where(Eq(fk, v))
	}
	q.Select(s.Select...)
	q.Rows = rows
	q.setPage(page)
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
* RollupOperation: Specifies how the result of a rollup query is applied to each resulting row.
**/
type RollupOperation string

const (
	RollupCount  RollupOperation = "count"
	RollupSum    RollupOperation = "sum"
	RollupAvg    RollupOperation = "avg"
	RollupMin    RollupOperation = "min"
	RollupMax    RollupOperation = "max"
	RollupRow    RollupOperation = "row"
	RollupObject RollupOperation = "object"
)

/**
* IsAggregate: Returns true if the operation is a SQL aggregate (count, sum, avg, min, max).
* @return bool
**/
func (s RollupOperation) IsAggregate() bool {
	switch s {
	case RollupCount, RollupSum, RollupAvg, RollupMin, RollupMax:
		return true
	}
	return false
}

/**
* IsValid: Returns true if the operation is one of the defined rollup operations.
* @return bool
**/
func (s RollupOperation) IsValid() bool {
	return s.IsAggregate() || s == RollupRow || s == RollupObject
}

/**
* Rollups: Defines a lookup against another model, executed after the main query for each resulting row.
* Keys maps a field of the resulting row to a field of the To model; Select lists the To fields
* and Operation defines how the result is applied (aggregate, merged row or nested object).
**/
type Rollups struct {
	To        *From             `json:"to"`
	Keys      map[string]string `json:"keys"`
	Select    []string          `json:"select"`
	Operation RollupOperation   `json:"operation"`
}

/**
* newRollup: Constructs a Rollups linking to the given model with join keys, selects and operation.
* @param to *Model, keys map[string]string, selects []string, operation RollupOperation
* @return *Rollups
**/
func newRollup(to *Model, keys map[string]string, selects []string, operation RollupOperation) *Rollups {
	return &Rollups{
		To:        getFrom(to, ""),
		Keys:      keys,
		Select:    selects,
		Operation: operation,
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
