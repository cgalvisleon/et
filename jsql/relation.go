package jsql

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
)

func detailKeys(key, foreignKey string) map[string]string {
	return map[string]string{
		key: foreignKey,
	}
}

func rollupKeys(key, foreignKey string) map[string]string {
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
* ref: Returns the reference of the detail.
* @return et.Json
**/
func (s *Detail) ref() et.Json {
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
* getQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *Detail) getQuery(item et.Json, page, rows int) *Query {
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
* isAggregate: Returns true if the operation is a SQL aggregate (count, sum, avg, min, max).
* @return bool
**/
func (s RollupOperation) isAggregate() bool {
	switch s {
	case RollupCount, RollupSum, RollupAvg, RollupMin, RollupMax:
		return true
	}
	return false
}

/**
* isValid: Returns true if the operation is one of the defined rollup operations.
* @return bool
**/
func (s RollupOperation) isValid() bool {
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
	Where     []*et.Condition   `json:"where"`
}

/**
* rollupJson: JSON shape of Rollups and QueryRollups; the conditions go as a where list
* ([{field: {op: value}}, {and: {...}}]), the same one a query takes.
**/
type rollupJson struct {
	To        *From             `json:"to"`
	Keys      map[string]string `json:"keys"`
	Select    []string          `json:"select"`
	Operation RollupOperation   `json:"operation"`
	Where     []et.Json         `json:"where"`
}

/**
* whereToJson: Returns the conditions as a where list.
* @param where []*et.Condition
* @return []et.Json
**/
func whereToJson(where []*et.Condition) []et.Json {
	result := make([]et.Json, 0, len(where))
	for _, cond := range where {
		if cond != nil {
			result = append(result, cond.ToJson())
		}
	}
	return result
}

/**
* rollupWhere: Validates the conditions of a rollup: they are joined to its keys with AND, and the drivers
* do not group conditions, so an OR would drop the key filter. The first one may come without a connector.
* @param where []*et.Condition
* @return []*et.Condition, error
**/
func rollupWhere(where []*et.Condition) ([]*et.Condition, error) {
	result := make([]*et.Condition, 0, len(where))
	for _, cond := range where {
		if cond == nil {
			continue
		}
		if cond.Connector == et.OR {
			return nil, errors.New(MSG_ROLLUP_WHERE_OR)
		}
		result = append(result, cond)
	}
	return result, nil
}

/**
* MarshalJSON: Writes the rollup with its conditions as a where list.
* @return []byte, error
**/
func (s Rollups) MarshalJSON() ([]byte, error) {
	return json.Marshal(rollupJson{To: s.To, Keys: s.Keys, Select: s.Select, Operation: s.Operation, Where: whereToJson(s.Where)})
}

/**
* UnmarshalJSON: Reads a rollup whose conditions come as a where list.
* @param data []byte
* @return error
**/
func (s *Rollups) UnmarshalJSON(data []byte) error {
	var v rollupJson
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	where, err := et.ToWheRes(v.Where)
	if err != nil {
		return err
	}
	*s = Rollups{To: v.To, Keys: v.Keys, Select: v.Select, Operation: v.Operation, Where: where}
	return nil
}

/**
* rowFields: Returns the fields of the resulting row the rollup needs: its keys and the
* fields its conditions reference with {field: "..."} (the first segment of a path).
* @return []string
**/
func (s *Rollups) rowFields() []string {
	result := make([]string, 0, len(s.Keys))
	for k := range s.Keys {
		result = append(result, k)
	}
	for _, cond := range s.Where {
		if ref, ok := rowRef(cond); ok {
			name, _, _ := strings.Cut(ref.Name, "->")
			if !slices.Contains(result, name) {
				result = append(result, name)
			}
		}
	}
	return result
}

/**
* queryRollups: Returns the rollup resolved for a query.
* @return *QueryRollups
**/
func (s *Rollups) queryRollups() *QueryRollups {
	return &QueryRollups{
		To:        s.To,
		Keys:      s.Keys,
		Select:    s.Select,
		Operation: s.Operation,
		Where:     s.Where,
	}
}

/**
* rowRef: Returns the reference of a condition whose value is {field: "..."}; in a rollup it names a field
* of the resulting row.
* @param cond *et.Condition
* @return et.Field, bool
**/
func rowRef(cond *et.Condition) (et.Field, bool) {
	if cond == nil || cond.Value.Type != et.FIELD {
		return et.Field{}, false
	}
	ref, ok := cond.Value.Value.(et.Field)
	return ref, ok
}

/**
* newRollup: Constructs a Rollups linking to the given model with join keys, selects, operation and conditions.
* @param to *Model, keys map[string]string, selects []string, operation RollupOperation, where []*et.Condition
* @return *Rollups
**/
func newRollup(to *Model, keys map[string]string, selects []string, operation RollupOperation, where []*et.Condition) *Rollups {
	return &Rollups{
		To:        getFrom(to, ""),
		Keys:      keys,
		Select:    selects,
		Operation: operation,
		Where:     where,
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

type Master struct {
	From   *From             `json:"from"`
	To     *From             `json:"to"`
	Bridge *From             `json:"bridge"`
	Keys   map[string]string `json:"keys"`
	ToKeys map[string]string `json:"to_keys"`
	Select []string          `json:"select"`
	Rows   int               `json:"rows"`
}

/**
* ref: Returns the reference of the master.
* @return et.Json
**/
func (s *Master) ref() et.Json {
	return et.Json{
		"to":     s.To,
		"bridge": s.Bridge,
	}
}

/**
* init: Initializes the master.
* @return error
**/
func (s *Master) init() error {
	if s.To == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if s.To.Model == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if s.Bridge == nil {
		return errors.New(MSG_BRIDGE_MODEL_REQUIRED)
	}

	if s.Bridge.Model == nil {
		return errors.New(MSG_BRIDGE_MODEL_REQUIRED)
	}

	// The bridge has foreign keys to both models, so both tables must exist first.
	if s.From != nil && s.From.Model != nil {
		if err := s.From.Model.Init(); err != nil {
			return err
		}
	}

	err := s.To.Model.Init()
	if err != nil {
		return err
	}

	err = s.Bridge.Model.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* newMaster: Creates a new master.
* @param from, to, bridge *Model, keys, toKeys map[string]string, selects []string
* @return *Master
**/
func newMaster(from, to, bridge *Model, keys, toKeys map[string]string, selects []string) *Master {
	return &Master{
		From:   getFrom(from, ""),
		To:     getFrom(to, ""),
		Bridge: getFrom(bridge, ""),
		Keys:   keys,
		ToKeys: toKeys,
		Select: selects,
	}
}
