package jsql

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
)

/**
* detailKeys: The keys of a DETAIL relation: the parent key mapped to the foreign key of the detail.
* @param key, foreignKey string
* @return map[string]string
**/
func detailKeys(key, foreignKey string) map[string]string {
	return map[string]string{
		key: foreignKey,
	}
}

/**
* rollupKeys: The keys of a ROLLUP relation: the foreign key of the model mapped to the key of the related model.
* @param key, foreignKey string
* @return map[string]string
**/
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
	detail := &QueryDetail{To: s.To, Keys: s.Keys, Select: s.Select, Page: page, Rows: rows}
	return detail.getQuery(item)
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
	OrdersBy  []*Index          `json:"order_by"`
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
	OrdersBy  []*Index          `json:"order_by"`
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
* rollupWhere: Returns the conditions of a rollup without the nil ones; the first may come without a connector.
* @param where []*et.Condition
* @return []*et.Condition
**/
func rollupWhere(where []*et.Condition) []*et.Condition {
	result := make([]*et.Condition, 0, len(where))
	for _, cond := range where {
		if cond != nil {
			result = append(result, cond)
		}
	}
	return result
}

/**
* rollupConditions: Repeats the keys in every OR term of the where, so an OR cannot drop them (copies).
* @param keys, where []*et.Condition
* @return []*et.Condition
**/
func rollupConditions(keys, where []*et.Condition) []*et.Condition {
	terms := [][]*et.Condition{}
	for _, cond := range where {
		if len(terms) == 0 || cond.Connector == et.OR {
			terms = append(terms, []*et.Condition{})
		}
		terms[len(terms)-1] = append(terms[len(terms)-1], cond)
	}
	if len(terms) == 0 {
		terms = append(terms, []*et.Condition{})
	}

	result := make([]*et.Condition, 0, len(terms)*(len(keys)+1)+len(where))
	for _, term := range terms {
		for i, cond := range append(slices.Clone(keys), term...) {
			tmp := *cond
			switch {
			case len(result) == 0:
				tmp.Connector = et.NaC
			case i == 0:
				tmp.Connector = et.OR
			default:
				tmp.Connector = et.AND
			}
			result = append(result, &tmp)
		}
	}
	return result
}

/**
* MarshalJSON: Writes the rollup with its conditions as a where list.
* @return []byte, error
**/
func (s Rollups) MarshalJSON() ([]byte, error) {
	return json.Marshal(rollupJson{To: s.To, Keys: s.Keys, Select: s.Select, Operation: s.Operation, Where: whereToJson(s.Where), OrdersBy: s.OrdersBy})
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
	*s = Rollups{To: v.To, Keys: v.Keys, Select: v.Select, Operation: v.Operation, Where: where, OrdersBy: v.OrdersBy}
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
		OrdersBy:  s.OrdersBy,
	}
}

/**
* orderBy: Adds an order to the rollup, which picks the first record for RollupRow and RollupObject
* (without one the database returns any). sorted true is ascending (the default).
* @param field string, sorted ...bool
* @return *Rollups
**/
func (s *Rollups) orderBy(field string, sorted ...bool) *Rollups {
	asc := true
	if len(sorted) > 0 {
		asc = sorted[0]
	}
	// A new slice: queries resolved before keep the one they took.
	s.OrdersBy = append(slices.Clone(s.OrdersBy), &Index{Name: field, Sorted: asc})
	return s
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

// batchKey is the alias under which a batch query returns the key of the parent row; it is removed from the result.
const batchKey = "_jsql_key"

// batchSize is how many key values go in one IN of a batch query (Oracle accepts up to 1000).
const batchSize = 500

/**
* valueKey: Returns the text a key value is grouped by, so that the same value read from the parent row
* and from the related row match whatever its Go type (1 and 1.0 give "1").
* @param v any
* @return string
**/
func valueKey(v any) string {
	return fmt.Sprint(v)
}

/**
* singleKey: Returns the only key of a relation; batches are resolved only for relations with one key.
* @param keys map[string]string
* @return string, string, bool
**/
func singleKey(keys map[string]string) (string, string, bool) {
	if len(keys) != 1 {
		return "", "", false
	}
	for k, fk := range keys {
		return k, fk, true
	}
	return "", "", false
}

/**
* hasKeys: Returns true when item has a value for every key of a relation.
* @param item et.Json, keys map[string]string
* @return bool
**/
func hasKeys(item et.Json, keys map[string]string) bool {
	for k := range keys {
		if v, ok := item[k]; !ok || v == nil {
			return false
		}
	}
	return true
}

/**
* keyValues: Returns the distinct, non-null values of field k in items.
* @param items []et.Json, k string
* @return []any
**/
func keyValues(items []et.Json, k string) []any {
	seen := map[string]bool{}
	result := make([]any, 0, len(items))
	for _, item := range items {
		v, ok := item[k]
		if !ok || v == nil {
			continue
		}
		key := valueKey(v)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}

/**
* batchRows: Runs the query that build returns for each chunk of values (batchSize per IN) and joins the rows.
* @param tx *Tx, values []any, build func(chunk []any) *Query
* @return []et.Json, error
**/
func batchRows(tx *Tx, values []any, build func(chunk []any) *Query) ([]et.Json, error) {
	result := make([]et.Json, 0)
	for i := 0; i < len(values); i += batchSize {
		end := min(i+batchSize, len(values))
		items, err := build(values[i:end]).AllTx(tx)
		if err != nil {
			return nil, err
		}
		result = append(result, items.Result...)
	}
	return result, nil
}

/**
* groupRows: Groups rows by the value of field; with remove, field is deleted from each row.
* @param rows []et.Json, field string, remove bool
* @return map[string][]et.Json
**/
func groupRows(rows []et.Json, field string, remove bool) map[string][]et.Json {
	result := map[string][]et.Json{}
	for _, row := range rows {
		key := valueKey(row[field])
		if remove {
			delete(row, field)
		}
		result[key] = append(result[key], row)
	}
	return result
}

/**
* pageOf: Returns the rows of a 1-based page of rows records; rows <= 0 returns them all.
* @param list []et.Json, page, rows int
* @return []et.Json
**/
func pageOf(list []et.Json, page, rows int) []et.Json {
	if rows <= 0 {
		return list
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * rows
	if start >= len(list) {
		return []et.Json{}
	}
	return list[start:min(start+rows, len(list))]
}

/**
* firstOf: Returns the first row of list, or nil.
* @param list []et.Json
* @return et.Json
**/
func firstOf(list []et.Json) et.Json {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

/**
* setRelations: Resolves details, masters and rollups; with several rows, single-key relations go in batches.
* @param tx *Tx, items []et.Json
* @return error
**/
func (s *Query) setRelations(tx *Tx, items []et.Json) error {
	if len(items) == 0 {
		return nil
	}
	for name, detail := range s.Details {
		if err := s.setDetail(tx, items, name, detail); err != nil {
			return err
		}
	}
	for name, master := range s.Masters {
		if err := s.setMaster(tx, items, name, master); err != nil {
			return err
		}
	}
	for name, rollup := range s.Rollups {
		if err := s.setRollup(tx, items, name, rollup); err != nil {
			return err
		}
	}
	return nil
}

/**
* rowsOf: Returns how many records a detail or master shows per parent: its Rows, or MaxRows when not set.
* @param rows int
* @return int
**/
func (s *Query) rowsOf(rows int) int {
	if rows <= 0 {
		return s.MaxRows
	}
	return rows
}

/**
* setDetail: Sets item[name] to the list of related records of each item.
* @param tx *Tx, items []et.Json, name string, detail *QueryDetail
* @return error
**/
func (s *Query) setDetail(tx *Tx, items []et.Json, name string, detail *QueryDetail) error {
	k, fk, ok := singleKey(detail.Keys)
	if !ok || len(items) == 1 {
		for _, item := range items {
			if !hasKeys(item, detail.Keys) {
				// Without its keys the query would have no filter and bring every record.
				item[name] = []et.Json{}
				continue
			}
			result, err := detail.GetQuery(item).AllTx(tx)
			if err != nil {
				return err
			}
			item[name] = result.Result
		}
		return nil
	}

	field := fk
	rows, err := batchRows(tx, keyValues(items, k), func(chunk []any) *Query {
		q := NewQuery(detail.To.Model, "A")
		q.Where(In(fk, chunk))
		if len(detail.Select) > 0 {
			field = batchKey
			q.Select(detail.Select...)
			q.Select(fk + ":" + batchKey)
		}
		q.unlimited = true
		return q
	})
	if err != nil {
		return err
	}

	groups := groupRows(rows, field, field == batchKey)
	for _, item := range items {
		list := []et.Json{}
		if v, ok := item[k]; ok && v != nil {
			list = pageOf(groups[valueKey(v)], detail.Page, s.rowsOf(detail.Rows))
		}
		item[name] = list
	}
	return nil
}

/**
* setMaster: Sets item[name] to the records linked to each item through the bridge: one object (or nil)
* when the master shows one record, a list otherwise.
* @param tx *Tx, items []et.Json, name string, master *QueryDetail
* @return error
**/
func (s *Query) setMaster(tx *Tx, items []et.Json, name string, master *QueryDetail) error {
	assign := func(item et.Json, list []et.Json) {
		if master.Rows == 1 {
			// 1 to 1 relation: the row gets the linked record as an object.
			item[name] = firstOf(list)
			return
		}
		item[name] = list
	}

	k, fk, ok := singleKey(master.Keys)
	if !ok || len(items) == 1 || len(master.Select) == 0 || master.Bridge == nil || master.Bridge.Model == nil {
		for _, item := range items {
			if !hasKeys(item, master.Keys) {
				assign(item, []et.Json{})
				continue
			}
			result, err := master.GetQuery(item).AllTx(tx)
			if err != nil {
				return err
			}
			assign(item, result.Result)
		}
		return nil
	}

	rows, err := batchRows(tx, keyValues(items, k), func(chunk []any) *Query {
		q := NewQuery(master.To.Model, "A")
		on := make([]*et.Condition, 0, len(master.ToKeys))
		for tk, tfk := range master.ToKeys {
			on = append(on, Eq("B."+tfk, "A."+tk))
		}
		q.Join(master.Bridge.Model, "B", on)
		q.Where(In("B."+fk, chunk))
		q.Select(master.Select...)
		q.Select("B." + fk + ":" + batchKey)
		q.unlimited = true
		return q
	})
	if err != nil {
		return err
	}

	groups := groupRows(rows, batchKey, true)
	for _, item := range items {
		list := []et.Json{}
		if v, ok := item[k]; ok && v != nil {
			list = pageOf(groups[valueKey(v)], 1, s.rowsOf(master.Rows))
		}
		assign(item, list)
	}
	return nil
}

/**
* rollupValue: Applies the result of a rollup to item[name]: the value of its single selected field, or the
* record when it selects several (or none, for RollupObject).
* @param item et.Json, name string, rollup *QueryRollups, first et.Json
**/
func rollupValue(item et.Json, name string, rollup *QueryRollups, first et.Json) {
	if rollup.Operation != RollupObject && len(rollup.Select) == 1 {
		var val any
		for _, v := range first {
			val = v
		}
		item[name] = val
		return
	}
	item[name] = first
}

/**
* setRollupRow: Resolves a rollup for one item with its own query.
* @param tx *Tx, item et.Json, name string, rollup *QueryRollups
* @return error
**/
func setRollupRow(tx *Tx, item et.Json, name string, rollup *QueryRollups) error {
	qry, ok := rollup.GetQuery(item)
	if !ok {
		return nil
	}

	if rollup.Operation == RollupCount && len(rollup.Select) == 0 {
		count, err := qry.CountTx(tx)
		if err != nil {
			return err
		}
		item[name] = count
		return nil
	}

	result, err := qry.AllTx(tx)
	if err != nil {
		return err
	}
	rollupValue(item, name, rollup, firstOf(result.Result))
	return nil
}

/**
* hasRowRefs: Returns true when the where of the rollup reads fields of the row ({field: "..."}).
* @return bool
**/
func (s *QueryRollups) hasRowRefs() bool {
	for _, cond := range s.Where {
		if _, ok := rowRef(cond); ok {
			return true
		}
	}
	return false
}

/**
* setRollup: Sets item[name] to the aggregate or the first related record; a batch groups by the key.
* @param tx *Tx, items []et.Json, name string, rollup *QueryRollups
* @return error
**/
func (s *Query) setRollup(tx *Tx, items []et.Json, name string, rollup *QueryRollups) error {
	k, fk, ok := singleKey(rollup.Keys)
	if !ok || len(items) == 1 || rollup.hasRowRefs() || rollup.To == nil || rollup.To.Model == nil {
		for _, item := range items {
			if err := setRollupRow(tx, item, name, rollup); err != nil {
				return err
			}
		}
		return nil
	}

	aggregate := rollup.Operation.IsAggregate()
	count := rollup.Operation == RollupCount && len(rollup.Select) == 0
	aliases := make([]string, 0, len(rollup.Select))
	field := fk
	rows, err := batchRows(tx, keyValues(items, k), func(chunk []any) *Query {
		q := NewQuery(rollup.To.Model, "A")
		q.addCondition(rollupConditions([]*et.Condition{In(fk, chunk)}, rollup.Where))
		switch {
		case count:
			field = batchKey
			q.Select(fk+":"+batchKey, fmt.Sprintf("count(%s):%s", fk, name))
			q.GroupBy(fk)
		case aggregate:
			field = batchKey
			q.Select(fk + ":" + batchKey)
			aliases = aliases[:0]
			for _, sel := range rollup.Select {
				agg := rollupAggSelect(rollup.Operation, sel)
				aliases = append(aliases, agg[strings.LastIndex(agg, ":")+1:])
				q.Select(agg)
			}
			q.GroupBy(fk)
		case len(rollup.Select) > 0:
			field = batchKey
			q.Select(rollup.Select...)
			q.Select(fk + ":" + batchKey)
		}
		if !aggregate {
			// The rows of each parent keep this order when they are grouped: the first one is the rollup's.
			for _, order := range rollup.OrdersBy {
				q.OrderBy(order.Name, order.Sorted)
			}
		}
		q.unlimited = true
		return q
	})
	if err != nil {
		return err
	}

	groups := groupRows(rows, field, field == batchKey)
	for _, item := range items {
		v, ok := item[k]
		if !ok {
			continue
		}
		var first et.Json
		if v != nil {
			first = firstOf(groups[valueKey(v)])
		}
		switch {
		case count:
			item[name] = 0
			if first != nil {
				item[name] = first.Int(name)
			}
		case aggregate:
			if first == nil {
				// Without records: count and sum give 0, avg, min and max null.
				first = et.Json{}
				for _, alias := range aliases {
					first[alias] = nil
					if rollup.Operation == RollupCount || rollup.Operation == RollupSum {
						first[alias] = 0
					}
				}
			}
			rollupValue(item, name, rollup, first)
		default:
			rollupValue(item, name, rollup, first)
		}
	}
	return nil
}

/**
* QueryDetail: Defines a relationship to another model, including join keys and cascade rules.
**/
type QueryDetail struct {
	To     *From             `json:"to"`
	Bridge *From             `json:"bridge,omitempty"`
	ToKeys map[string]string `json:"to_keys,omitempty"`
	Keys   map[string]string `json:"keys"`
	Select []string          `json:"select"`
	Page   int               `json:"page"`
	Rows   int               `json:"rows"`
}

/**
* QueryRollups: Rollup resolved for a query; executed per resulting row after the main SQL.
**/
type QueryRollups struct {
	To        *From             `json:"to"`
	Keys      map[string]string `json:"keys"`
	Select    []string          `json:"select"`
	Operation RollupOperation   `json:"operation"`
	Where     []*et.Condition   `json:"where"`
	OrdersBy  []*Index          `json:"order_by"`
}

/**
* MarshalJSON: Writes the rollup with its conditions as a where list.
* @return []byte, error
**/
func (s QueryRollups) MarshalJSON() ([]byte, error) {
	return json.Marshal(rollupJson{To: s.To, Keys: s.Keys, Select: s.Select, Operation: s.Operation, Where: whereToJson(s.Where), OrdersBy: s.OrdersBy})
}

/**
* UnmarshalJSON: Reads a rollup whose conditions come as a where list.
* @param data []byte
* @return error
**/
func (s *QueryRollups) UnmarshalJSON(data []byte) error {
	var v Rollups
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = *v.queryRollups()
	return nil
}

/**
* resolveWhere: Returns the conditions of the rollup for item: a {field: "..."} value is read from item.
* Returns false when item lacks a referenced field, so the rollup is not applied (as with a missing key).
* @param item et.Json
* @return []*et.Condition, bool
**/
func (s *QueryRollups) resolveWhere(item et.Json) ([]*et.Condition, bool) {
	result := make([]*et.Condition, 0, len(s.Where))
	for _, cond := range s.Where {
		if cond == nil {
			continue
		}
		tmp := *cond
		if ref, ok := rowRef(cond); ok {
			name, _, _ := strings.Cut(ref.Name, "->")
			if _, exists := item[name]; !exists {
				return nil, false
			}
			tmp.Value = et.NewValue(ref.ValueOf(item))
		}
		result = append(result, &tmp)
	}
	return result, true
}

/**
* rollupAggSelect: Wraps a select field as an aggregate expression "op(field):as" understood by GetField.
* @param op RollupOperation, field string
* @return string
**/
func rollupAggSelect(op RollupOperation, field string) string {
	name, as := field, field
	if i := strings.LastIndex(field, ":"); i != -1 {
		name, as = field[:i], field[i+1:]
	}
	if i := strings.LastIndex(as, "."); i != -1 {
		as = as[i+1:]
	}
	as = strings.ReplaceAll(as, "->", "_")
	return fmt.Sprintf("%s(%s):%s", op, name, as)
}

/**
* getQuery: Returns the query for the rollup filtered by the keys of the given row.
* Returns false when the row lacks any of the keys, so the rollup is not applied.
* @param item et.Json
* @return *Query, bool
**/
func (s *QueryRollups) getQuery(item et.Json) (*Query, bool) {
	if s.To == nil || s.To.Model == nil {
		return nil, false
	}

	where, ok := s.resolveWhere(item)
	if !ok {
		return nil, false
	}

	keys := make([]*et.Condition, 0, len(s.Keys))
	for k, fk := range s.Keys {
		v, exists := item[k]
		if !exists {
			return nil, false
		}
		keys = append(keys, Eq(fk, v))
	}
	q := NewQuery(s.To.Model, "A")
	q.addCondition(rollupConditions(keys, where))

	if s.Operation.IsAggregate() {
		for _, field := range s.Select {
			q.Select(rollupAggSelect(s.Operation, field))
		}
		return q, true
	}

	q.Select(s.Select...)
	for _, order := range s.OrdersBy {
		q.OrderBy(order.Name, order.Sorted)
	}
	q.Rows = 1
	return q, true
}

/**
* getQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *QueryDetail) getQuery(item et.Json) *Query {
	q := NewQuery(s.To.Model, "A")
	if s.Bridge != nil && s.Bridge.Model != nil {
		// Master: the To rows linked to the item through the bridge (A = To, B = bridge).
		on := make([]*et.Condition, 0, len(s.ToKeys))
		for k, fk := range s.ToKeys {
			on = append(on, Eq("B."+fk, "A."+k))
		}
		q.Join(s.Bridge.Model, "B", on)
		for k, fk := range s.Keys {
			q.Where(Eq("B."+fk, item[k]))
		}
		q.Select(s.Select...)
		q.Rows = s.Rows
		return q
	}
	for k, fk := range s.Keys {
		v, exists := item[k]
		if !exists {
			continue
		}
		q.Where(Eq(fk, v))
	}
	q.Select(s.Select...)
	q.Rows = s.Rows
	q.setPage(s.Page)
	return q
}
