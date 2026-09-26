package jsql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
)

/**
* From: Identifies a table source with its fully-qualified name and SQL alias.
**/
type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Table    string `json:"table"`
	As       string `json:"as"`
	Model    *Model `json:"-"`
}

/**
* ref: Returns the reference of the from.
* @return et.Json
**/
func (s *From) ref() et.Json {
	return et.Json{
		"database": s.Database,
		"schema":   s.Schema,
		"name":     s.Name,
		"table":    s.Table,
		"as":       s.As,
	}
}

/**
* getFrom: Builds a From descriptor from a model, using as as the SQL alias (defaults to table name).
* @param model *Model, as string
* @return *From
**/
func getFrom(model *Model, as string) *From {
	if as == "" {
		as = model.Table
	}
	return &From{
		Database: model.database,
		Schema:   model.Schema,
		Name:     model.Name,
		Table:    model.Table,
		As:       as,
		Model:    model,
	}
}

/**
* Field: A parsed field reference (et.Field) resolved against a model: column type, data type and origin.
**/
type Field struct {
	et.Field
	TypeColumn TypeColumn  `json:"type_column"`
	TypeData   et.TypeData `json:"type_data"`
	From       *From       `json:"from"`
}

/**
* JoinType: Specifies the SQL join strategy for a Join clause.
**/
type JoinType string

const (
	INNER_JOIN JoinType = "inner"
	LEFT_JOIN  JoinType = "left"
	RIGHT_JOIN JoinType = "right"
	FULL_JOIN  JoinType = "full"
)

/**
* Join: Represents a JOIN clause with its type, target table, and ON conditions.
**/
type Join struct {
	Type      JoinType        `json:"type"`
	To        *From           `json:"to"`
	Condition []*et.Condition `json:"condition"`
	query     *Query          `json:"-"`
}

/**
* newJoin: Constructs a Join entry linked to its parent query.
* @param query *Query, typ JoinType, to *From, conditions []*et.Condition
* @return *Join
**/
func newJoin(query *Query, typ JoinType, to *From, conditions []*et.Condition) *Join {
	return &Join{
		Type:      typ,
		To:        to,
		Condition: conditions,
		query:     query,
	}
}

/**
* QuerySection: Tracks which clause is currently active for And/Or routing.
**/
type QuerySection int

const (
	whereSection QuerySection = iota
	joinSection
	havingSection
)

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

	q := NewQuery(s.To.Model, "A")
	for k, fk := range s.Keys {
		v, exists := item[k]
		if !exists {
			return nil, false
		}
		q.Where(Eq(fk, v))
	}

	if s.Operation.IsAggregate() {
		for _, field := range s.Select {
			q.Select(rollupAggSelect(s.Operation, field))
		}
		return q, true
	}

	q.Select(s.Select...)
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

type Calc struct {
	Model  *Model `json:"model"`
	Module string `json:"module"`
}

/**
* Query: Holds all clauses needed to build a SELECT statement.
**/
type Query struct {
	ID             string                   `json:"id"`
	Froms          []*From                  `json:"froms"`
	Joins          []*Join                  `json:"joins"`
	Selects        []string                 `json:"selects"`
	Conditions     []*et.Condition          `json:"conditions"`
	Hiddens        []string                 `json:"hidden"`
	GroupsBy       []string                 `json:"group_by"`
	OrdersBy       []*Index                 `json:"order_by"`
	Havings        []*et.Condition          `json:"havings"`
	Offset         int                      `json:"offset"`
	Rows           int                      `json:"rows"`
	UseSourceField bool                     `json:"use_source_field"`
	Details        map[string]*QueryDetail  `json:"details"`
	Masters        map[string]*QueryDetail  `json:"masters"`
	Rollups        map[string]*QueryRollups `json:"rollups"`
	CalcFuns       map[string]CalcFunction  `json:"calc_funs"`
	Calcs          map[string]*Calc         `json:"calcs"`
	IsExists       bool                     `json:"is_exists"`
	IsCount        bool                     `json:"is_count"`
	section        QuerySection             `json:"-"`
	MaxRows        int                      `json:"-"`
	db             *DB                      `json:"-"`
	isDebug        bool                     `json:"-"`
	isTest         bool                     `json:"-"`
	relationKeys   []string                 `json:"-"`
	err            error                    `json:"-"`
}

/**
* newQuery: Creates a Query with the model as the primary FROM source.
* @param model *Model, as ...string
* @return *Query
**/
func newQuery(model *Model, as ...string) *Query {
	if len(as) == 0 || as[0] == "" {
		as = []string{"A"}
	}
	result := &Query{
		ID:         reg.UUID(),
		Froms:      make([]*From, 0),
		Joins:      make([]*Join, 0),
		Selects:    make([]string, 0),
		Conditions: make([]*et.Condition, 0),
		Hiddens:    make([]string, 0),
		GroupsBy:   make([]string, 0),
		OrdersBy:   make([]*Index, 0),
		Havings:    make([]*et.Condition, 0),
		Details:    make(map[string]*QueryDetail, 0),
		Masters:    make(map[string]*QueryDetail, 0),
		Rollups:    make(map[string]*QueryRollups, 0),
		CalcFuns:   make(map[string]CalcFunction, 0),
		Calcs:      make(map[string]*Calc, 0),
		section:    whereSection,
		MaxRows:    model.db.RecordLimit,
		db:         model.db,
		isDebug:    model.IsDebug,
	}
	result.addFrom(model, as[0])
	return result
}

/**
* serialize: Marshals the query metadata to JSON bytes.
* @return []byte, error
**/
func (s *Query) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* toJson: Returns the query metadata as an et.Json map.
* @return et.Json
**/
func (s *Query) toJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* setDebug: Sets the debug flag for the query.
* @param debug bool
* @return *Query
**/
func (s *Query) setDebug(debug bool) *Query {
	s.isDebug = debug
	return s
}

/**
* debug: Enables SQL logging for this query and returns it for chaining.
* @return *Query
**/
func (s *Query) debug() *Query {
	return s.setDebug(true)
}

/**
* test: Enables test mode — SQL is generated but not executed.
* @return *Query
**/
func (s *Query) test() *Query {
	s.isTest = true
	return s
}

/**
* getField: Parses a field reference with et.ToField and resolves it against the query's origins.
* Supports "field", "field:as", "from.field", "from.field:as", "agg(field)", "agg(field):as",
* "field->a->b" and the "|page:n" suffix.
* @param field string
* @return (*Field, bool)
**/
func (s *Query) getField(field string) (*Field, bool) {
	def, ok := et.ToField(field)
	if !ok {
		return nil, false
	}

	from := s.getFrom(def.Source)
	if from == nil {
		return nil, false
	}

	col, ok := from.Model.GetColumn(def.Name)
	if !ok {
		return nil, false
	}

	return &Field{
		Field:      def,
		TypeColumn: col.TypeColumn,
		TypeData:   col.TypeData,
		From:       from,
	}, true
}

/**
* getFrom: Returns the origin (FROM or JOIN) whose name or alias matches name; an empty name returns the first FROM.
* @param name string
* @return *From
**/
func (s *Query) getFrom(name string) *From {
	if len(s.Froms) == 0 {
		return nil
	}
	if name == "" {
		return s.Froms[0]
	}
	for _, from := range s.Froms {
		if from.Name == name || from.As == name {
			return from
		}
	}
	for _, join := range s.Joins {
		if join.To != nil && (join.To.Name == name || join.To.As == name) {
			return join.To
		}
	}
	return nil
}

/**
* addFrom: Appends a FROM entry for the given model with the specified alias.
* @param model *Model, as string
* @return *Query
**/
func (s *Query) addFrom(model *Model, as string) *Query {
	from := getFrom(model, as)
	s.Froms = append(s.Froms, from)
	if !s.UseSourceField {
		s.UseSourceField = model.SourceField != ""
	}
	return s
}

/**
* join: Appends a JOIN clause of the given type with its ON condition.
* @param model *Model, as string, tp JoinType, conditions []*et.Condition
* @return *Join
**/
func (s *Query) join(model *Model, as string, tp JoinType, conditions []*et.Condition) *Join {
	result := newJoin(s, tp, getFrom(model, as), conditions)
	s.Joins = append(s.Joins, result)
	s.section = joinSection
	return result
}

/**
* joinInner: Appends an INNER JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) joinInner(model *Model, as string, on []*et.Condition) *Query {
	s.join(model, as, INNER_JOIN, on)
	return s
}

/**
* leftJoin: Appends a LEFT JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) leftJoin(model *Model, as string, on []*et.Condition) *Query {
	s.join(model, as, LEFT_JOIN, on)
	return s
}

/**
* rightJoin: Appends a RIGHT JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) rightJoin(model *Model, as string, on []*et.Condition) *Query {
	s.join(model, as, RIGHT_JOIN, on)
	return s
}

/**
* fullJoin: Appends a FULL JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) fullJoin(model *Model, as string, on []*et.Condition) *Query {
	s.join(model, as, FULL_JOIN, on)
	return s
}

/**
* addCondition: Appends a slice of conditions directly to the WHERE clause list.
* @param conds []*et.Condition
* @return *Query
**/
func (s *Query) addCondition(conds []*et.Condition) *Query {
	s.Conditions = append(s.Conditions, conds...)
	return s
}

/**
* selects: Appends fields to the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) selects(fields ...string) *Query {
	s.Selects = append(s.Selects, fields...)
	return s
}

/**
* calc: Appends fields to the DETAIL clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) calc(fields ...string) *Query {
	for _, field := range fields {
		for _, from := range s.Froms {
			fn, ok := from.Model.calcs[field]
			if !ok {
				continue
			}
			s.CalcFuns[field] = fn
		}
	}
	return s
}

/**
* detail: Appends fields to the DETAIL clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) detail(fields ...string) *Query {
	for _, field := range fields {
		from := s.Froms[0]
		if from == nil {
			continue
		}

		detail, ok := from.Model.Details[field]
		if !ok {
			continue
		}

		s.Details[field] = &QueryDetail{
			To:     detail.To,
			Keys:   detail.Keys,
			Select: detail.Select,
			Page:   1,
			Rows:   detail.Rows,
		}
	}

	return s
}

/**
* master: Appends fields to the MASTER clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) master(fields ...string) *Query {
	for _, field := range fields {
		from := s.Froms[0]
		if from == nil {
			continue
		}

		master, ok := from.Model.Masters[field]
		if !ok {
			continue
		}

		rows := master.Rows
		if rows <= 0 {
			rows = s.MaxRows
		}
		s.Masters[field] = &QueryDetail{
			To:     master.To,
			Bridge: master.Bridge,
			ToKeys: master.ToKeys,
			Keys:   master.Keys,
			Select: master.Select,
			Page:   1,
			Rows:   rows,
		}
	}
	return s
}

/**
* hidden: Appends fields to the HIDDEN clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) hidden(fields ...string) *Query {
	s.Hiddens = append(s.Hiddens, fields...)
	return s
}

/**
* addOneCondition: Appends a condition to the WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) addOneCondition(cond *et.Condition) *Query {
	if s.Conditions == nil {
		s.Conditions = []*et.Condition{}
	}
	if len(s.Conditions) > 0 && cond.Connector == et.AND {
		cond.Connector = et.AND
	}
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* where: Appends a condition to the WHERE clause and sets the active section to where.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) where(cond *et.Condition) *Query {
	s.AddCondition(cond)
	s.section = whereSection
	return s
}

/**
* and: Appends an AND condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) and(cond *et.Condition) *Query {
	cond.Connector = et.AND
	switch s.section {
	case joinSection:
		n := len(s.Joins)
		s.Joins[n-1].Condition = append(s.Joins[n-1].Condition, cond)
	case havingSection:
		s.Havings = append(s.Havings, cond)
	default:
		s.Conditions = append(s.Conditions, cond)
	}
	return s
}

/**
* or: Appends an OR condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) or(cond *et.Condition) *Query {
	cond.Connector = et.OR
	switch s.section {
	case joinSection:
		n := len(s.Joins)
		s.Joins[n-1].Condition = append(s.Joins[n-1].Condition, cond)
	case havingSection:
		s.Havings = append(s.Havings, cond)
	default:
		s.Conditions = append(s.Conditions, cond)
	}
	return s
}

/**
* groupBy: Adds one or more fields to the GROUP BY clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) groupBy(fields ...string) *Query {
	s.GroupsBy = append(s.GroupsBy, fields...)
	return s
}

/**
* having: Appends a condition to the HAVING clause and sets the active section to having.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) having(cond *et.Condition) *Query {
	s.Havings = append(s.Havings, cond)
	s.section = havingSection
	return s
}

/**
* setLimit: Sets the limit for the query.
* @param rows int
* @return *Query
**/
func (s *Query) setLimit(rows int) *Query {
	if rows > s.MaxRows {
		rows = s.MaxRows
	}
	s.Rows = rows
	return s
}

/**
* setPage: Sets the result offset based on the 1-based page number and current Rows limit.
* @param page int
* @return *Query
**/
func (s *Query) setPage(page int) *Query {
	s.Offset = (page - 1) * s.Rows
	return s
}

/**
* page: Sets the result offset based on the 1-based page number and current Rows limit.
* @param page int
* @return *Query
**/
func (s *Query) page(page int) *Query {
	return s.setPage(page)
}

/**
* orderBy: Appends a field to the ORDER BY clause; sorted=true means ASC, false means DESC.
* @param field string
* @param sorted bool
* @return *Query
**/
func (s *Query) orderBy(field string, sorted ...bool) *Query {
	sortedValue := true
	if len(sorted) > 0 {
		sortedValue = sorted[0]
	}
	s.OrdersBy = append(s.OrdersBy, &Index{Name: field, Sorted: sortedValue})
	return s
}

/**
* splitFromRef: Splits a from reference "schema.table:alias", "table:alias", "schema.table" or "table"
* into its parts; the schema is empty when not given.
* @param ref string
* @return schema, table, alias string, ok bool
**/
func splitFromRef(ref string) (schema, table, alias string, ok bool) {
	ref = strings.TrimSpace(ref)
	if i := strings.LastIndex(ref, ":"); i != -1 {
		ref, alias = ref[:i], ref[i+1:]
		if !fromIdent.MatchString(alias) {
			return "", "", "", false
		}
	}
	table = ref
	if i := strings.Index(ref, "."); i != -1 {
		schema, table = ref[:i], ref[i+1:]
		if !fromIdent.MatchString(schema) {
			return "", "", "", false
		}
	}
	if !fromIdent.MatchString(table) {
		return "", "", "", false
	}
	return schema, table, alias, true
}

var fromIdent = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

/**
* loadFrom: Applies the "from" of a JSON query. It is a reference ("schema.table:alias",
* "table:alias", "schema.table" or "table") or a list of them: the first one replaces the primary
* origin of the query and the others are added as more origins. Without schema, the schema of the
* model that runs the query is used; without alias, the primary origin is "A".
* @param value any
* @return error
**/
func (s *Query) loadFrom(value any) error {
	refs := []string{}
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		refs = append(refs, v)
	case []string:
		refs = v
	case []any:
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf(MSG_INVALID_FROM, fmt.Sprint(item))
			}
			refs = append(refs, str)
		}
	default:
		return fmt.Errorf(MSG_INVALID_FROM, fmt.Sprint(value))
	}
	if len(refs) == 0 || len(s.Froms) == 0 {
		return nil
	}

	defaultSchema := s.Froms[0].Schema
	froms := make([]*From, 0, len(refs))
	for i, ref := range refs {
		schema, table, alias, ok := splitFromRef(ref)
		if !ok {
			return fmt.Errorf(MSG_INVALID_FROM, ref)
		}
		if schema == "" {
			schema = defaultSchema
		}
		model, err := s.db.GetModel(schema, table)
		if err != nil {
			return fmt.Errorf(MSG_INVALID_FROM, ref)
		}
		if alias == "" && i == 0 {
			alias = "A"
		}
		froms = append(froms, getFrom(model, alias))
	}

	s.Froms = froms
	s.UseSourceField = false
	for _, from := range froms {
		if from.Model.SourceField != "" {
			s.UseSourceField = true
		}
	}
	return nil
}

/**
* addRelationKeys: When the query selects details, masters or rollups, adds the key fields they need
* to resolve each row (e.g. tp_doc for a rollup keyed by it) if they are not selected, and remembers
* them in relationKeys so they are removed from the result afterwards.
**/
func (s *Query) addRelationKeys() {
	if len(s.Selects) == 0 {
		return
	}
	selected := map[string]bool{}
	for _, field := range s.Selects {
		if fld, ok := s.GetField(field); ok {
			selected[fld.As] = true
		}
	}
	for _, field := range s.Selects {
		fld, ok := s.GetField(field)
		if !ok || fld.From == nil || fld.From.Model == nil {
			continue
		}
		model := fld.From.Model
		var keys map[string]string
		switch fld.TypeColumn {
		case DETAIL:
			if detail, ok := model.Details[fld.Name]; ok {
				keys = detail.Keys
			}
		case MASTER:
			if master, ok := model.Masters[fld.Name]; ok {
				keys = master.Keys
			}
		case ROLLUP:
			if rollup, ok := model.Rollups[fld.Name]; ok {
				keys = rollup.Keys
			}
		}
		for key := range keys {
			if selected[key] {
				continue
			}
			selected[key] = true
			name := key
			if len(s.Froms) > 1 {
				name = fld.From.As + "." + key
			}
			s.Selects = append(s.Selects, name)
			s.relationKeys = append(s.relationKeys, key)
		}
	}
}

/**
* setDetails: Sets the details for the query.
* @param tx *Tx
* @param item et.Json
* @return et.Json
**/
func (s *Query) setDetails(tx *Tx, item et.Json) et.Json {
	for name, detail := range s.Details {
		qry := detail.GetQuery(item)
		detailResult, err := qry.AllTx(tx)
		if err != nil {
			return item
		}
		item[name] = detailResult.Result
	}
	for name, master := range s.Masters {
		qry := master.GetQuery(item)
		masterResult, err := qry.AllTx(tx)
		if err != nil {
			return item
		}
		if master.Rows == 1 {
			// 1 to 1 relation: the row gets the linked record as an object.
			var first et.Json
			if len(masterResult.Result) > 0 {
				first = masterResult.Result[0]
			}
			item[name] = first
			continue
		}
		item[name] = masterResult.Result
	}
	return item
}

/**
* setRollup: Executes each rollup query for the row and applies its result by operation:
* aggregates set item[name] (a value for one select, an object for several),
* RollupRow sets item[name] to the value of its single selected field (or to the record when it selects
* several fields) and RollupObject sets item[name] to the resulting object.
* @param tx *Tx
* @param item et.Json
* @return et.Json
**/
func (s *Query) setRollup(tx *Tx, item et.Json) et.Json {
	for name, rollup := range s.Rollups {
		qry, ok := rollup.GetQuery(item)
		if !ok {
			continue
		}

		if rollup.Operation == RollupCount && len(rollup.Select) == 0 {
			count, err := qry.CountTx(tx)
			if err != nil {
				continue
			}
			item[name] = count
			continue
		}

		result, err := qry.AllTx(tx)
		if err != nil {
			continue
		}

		var first et.Json
		if result.Ok && len(result.Result) > 0 {
			first = result.Result[0]
		}

		switch rollup.Operation {
		case RollupRow:
			// The value of the selected field is assigned to the rollup attribute; with several
			// fields, the record is assigned as an object.
			if len(rollup.Select) == 1 {
				var val any
				for _, v := range first {
					val = v
				}
				item[name] = val
				continue
			}
			item[name] = first
		case RollupObject:
			item[name] = first
		default:
			if len(rollup.Select) == 1 {
				var val any
				for _, v := range first {
					val = v
				}
				item[name] = val
				continue
			}
			item[name] = first
		}
	}
	return item
}

/**
* setCalcFuns: Sets the calculations for the query.
* @param tx *Tx, item et.Json
**/
func (s *Query) setCalcFuns(tx *Tx, item et.Json) {
	for _, calc := range s.CalcFuns {
		calc(tx, item)
	}
}

/**
* setCalc: Runs the JS scripts of the CALC fields (defined with DefineCalc) over the item, which they
* receive as "item", and returns the item they leave.
* @param tx *Tx, item et.Json
**/
func (s *Query) setCalc(tx *Tx, item et.Json) (et.Json, error) {
	for name, calc := range s.Calcs {
		code := calc.Module
		if code == "" && calc.Model != nil {
			code = calc.Model.calcScripts[name]
		}
		if code == "" {
			continue
		}
		instance := jrex.NewInstance()
		instance.SetCode(code)
		calc.Model.wrapper(instance)
		setJsJson(instance, "item", item)
		instance.Set("tx", tx)
		_, err := instance.Run()
		if err != nil {
			return item, err
		}
		item = getJsJson(instance, "item")
	}

	return item, nil
}

/**
* allTx: Generates and executes a SELECT query inside the given transaction.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Query) allTx(tx *Tx) (et.Items, error) {
	if s.err != nil {
		return et.Items{}, s.err
	}
	if s.Rows == 0 {
		s.Rows = s.MaxRows
	}
	s.addRelationKeys()

	sql, err := s.db.query(s)
	if err != nil {
		return et.Items{}, err
	}

	if s.isDebug {
		logs.Debug("SQL:\n", sql)
	}

	if s.isTest {
		return et.Items{}, nil
	}

	result, err := s.db.SqlTx(tx, sql)
	if err != nil {
		return et.Items{}, err
	}

	for i, item := range result.Result {
		item = s.setDetails(tx, item)
		item = s.setRollup(tx, item)
		for _, key := range s.relationKeys {
			delete(item, key)
		}
		s.setCalcFuns(tx, item)
		item, err = s.setCalc(tx, item)
		if err != nil {
			return et.Items{}, err
		}
		result.Result[i] = item
	}

	return result, nil
}

/**
* all: Generates and executes a SELECT query without an explicit transaction.
* @return et.Items, error
**/
func (s *Query) all() (et.Items, error) {
	result, err := s.AllTx(nil)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* oneTx: Executes the query limited to one row inside the given transaction.
* @param tx *Tx
* @return et.Item, error
**/
func (s *Query) oneTx(tx *Tx) (et.Item, error) {
	s.Offset = 0
	s.Rows = 1
	result, err := s.AllTx(tx)
	if err != nil {
		return et.Item{}, err
	}

	if result.Ok {
		return result.One(1)
	}

	return et.Item{Result: et.Json{}}, nil
}

/**
* one: Executes the query limited to one row without an explicit transaction.
* @return et.Item, error
**/
func (s *Query) one() (et.Item, error) {
	return s.OneTx(nil)
}

/**
* firstTx: Executes the query limited to the first n rows inside the given transaction.
* @param tx *Tx, n int
* @return et.Items, error
**/
func (s *Query) firstTx(tx *Tx, n int) (et.Items, error) {
	return s.LimitTx(tx, 1, n)
}

/**
* first: Executes the query limited to the first n rows without an explicit transaction.
* @param n int
* @return et.Items, error
**/
func (s *Query) first(n int) (et.Items, error) {
	return s.FirstTx(nil, n)
}

/**
* limitTx: Sets the maximum number of rows to return.
* @param tx *Tx, page int, rows int
* @return et.Items, error
**/
func (s *Query) limitTx(tx *Tx, page, rows int) (et.Items, error) {
	s.setLimit(rows)
	s.setPage(page)
	return s.AllTx(tx)
}

/**
* limit: Sets the maximum number of rows to return.
* @param page int, rows int
* @return et.Items, error
**/
func (s *Query) limit(page, rows int) (et.Items, error) {
	return s.LimitTx(nil, page, rows)
}

/**
* existsTx: Returns the Model for the primary FROM source, or nil if not found.
* @return *Model
**/
func (s *Query) existsTx(tx *Tx) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	s.IsExists = true
	sql, err := s.db.query(s)
	if err != nil {
		return false, err
	}

	if s.isDebug {
		logs.Debug("SQL:\n", sql)
	}

	if s.isTest {
		return false, nil
	}

	result, err := s.db.SqlTx(tx, sql)
	if err != nil {
		return false, err
	}

	if !result.Ok {
		return false, nil
	}

	item, err := result.First()
	if err != nil {
		return false, err
	}

	exists := item.Bool("exists")
	return exists, nil
}

/**
* exists: Checks if any rows match the query conditions.
* @return bool, error
**/
func (s *Query) exists() (bool, error) {
	return s.ExistsTx(nil)
}

/**
* countTx: Executes the query and returns the count of matching rows within the given transaction.
* @param tx *Tx
* @return int, error
**/
func (s *Query) countTx(tx *Tx) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	s.IsCount = true
	sql, err := s.db.query(s)
	if err != nil {
		return 0, err
	}

	if s.isDebug {
		logs.Debug("SQL:\n", sql)
	}

	if s.isTest {
		return 0, nil
	}

	result, err := s.db.SqlTx(tx, sql)
	if err != nil {
		return 0, err
	}

	if !result.Ok {
		return 0, nil
	}

	item, err := result.First()
	if err != nil {
		return 0, err
	}

	count := item.Int("count")
	return count, nil
}

/**
* count: Executes the query and returns the count of matching rows.
* @return int, error
**/
func (s *Query) count() (int, error) {
	return s.CountTx(nil)
}

/**
* loadQuery: Loads a query from a JSON object.
* @param tx *Tx
* @param query et.Json
* @return et.Items, error
**/
func (s *Query) loadQuery(query et.Json) (*Query, error) {
	if err := s.loadFrom(query["from"]); err != nil {
		return s, err
	}

	join := query.ArrayJson("join")
	for _, js := range join {
		to := js.Str("to")
		as := ""
		args, ok := ArgWhitAs(to)
		if !ok {
			return s, fmt.Errorf(MSG_AS_REQUIRED_IN_JOIN, to)
		}
		to = args[0]
		as = args[1]
		args, ok = ArgWhitSchema(to)
		if !ok {
			return s, fmt.Errorf(MSG_INVALID_TO_IN_JOIN, to)
		}
		schema := args[0]
		table := args[1]
		modelTo, err := s.db.GetModel(schema, table)
		if err != nil {
			return s, fmt.Errorf(MSG_TO_REQUIRED_IN_JOIN, to)
		}

		on := js.ArrayJson("on")
		conditions, err := et.ToConditions(on)
		if err != nil {
			return s, err
		}
		s.join(modelTo, as, INNER_JOIN, conditions)
	}

	leftJoin := query.ArrayJson("left_join")
	for _, js := range leftJoin {
		to := js.Str("to")
		as := ""
		args, ok := ArgWhitAs(to)
		if !ok {
			return s, fmt.Errorf(MSG_AS_REQUIRED_IN_JOIN, to)
		}
		to = args[0]
		as = args[1]
		args, ok = ArgWhitSchema(to)
		if !ok {
			return s, fmt.Errorf(MSG_INVALID_TO_IN_JOIN, to)
		}
		schema := args[0]
		table := args[1]
		modelTo, err := s.db.GetModel(schema, table)
		if err != nil {
			return s, fmt.Errorf(MSG_TO_REQUIRED_IN_JOIN, to)
		}

		on := js.ArrayJson("on")
		conditions, err := et.ToConditions(on)
		if err != nil {
			return s, err
		}
		s.join(modelTo, as, LEFT_JOIN, conditions)
	}

	rightJoin := query.ArrayJson("right_join")
	for _, js := range rightJoin {
		to := js.Str("to")
		as := ""
		args, ok := ArgWhitAs(to)
		if !ok {
			return s, fmt.Errorf(MSG_AS_REQUIRED_IN_JOIN, to)
		}
		to = args[0]
		as = args[1]
		args, ok = ArgWhitSchema(to)
		if !ok {
			return s, fmt.Errorf(MSG_INVALID_TO_IN_JOIN, to)
		}
		schema := args[0]
		table := args[1]
		modelTo, err := s.db.GetModel(schema, table)
		if err != nil {
			return s, fmt.Errorf(MSG_TO_REQUIRED_IN_JOIN, to)
		}

		on := js.ArrayJson("on")
		conditions, err := et.ToConditions(on)
		if err != nil {
			return s, err
		}
		s.join(modelTo, as, RIGHT_JOIN, conditions)
	}

	fullJoin := query.ArrayJson("full_join")
	for _, js := range fullJoin {
		to := js.Str("to")
		as := ""
		args, ok := ArgWhitAs(to)
		if !ok {
			return s, fmt.Errorf(MSG_AS_REQUIRED_IN_JOIN, to)
		}
		to = args[0]
		as = args[1]
		args, ok = ArgWhitSchema(to)
		if !ok {
			return s, fmt.Errorf(MSG_INVALID_TO_IN_JOIN, to)
		}
		schema := args[0]
		table := args[1]
		modelTo, err := s.db.GetModel(schema, table)
		if err != nil {
			return s, fmt.Errorf(MSG_TO_REQUIRED_IN_JOIN, to)
		}

		on := js.ArrayJson("on")
		conditions, err := et.ToConditions(on)
		if err != nil {
			return s, err
		}
		s.join(modelTo, as, FULL_JOIN, conditions)
	}

	selects := query.ArrayStr("selects")
	if len(selects) > 0 {
		s.Select(selects...)
	}

	hiddens := query.ArrayStr("hiddens")
	if len(hiddens) > 0 {
		s.Hidden(hiddens...)
	}

	wheres := query.ArrayJson("where")
	conditions, err := et.ToConditions(wheres)
	if err != nil {
		return s, err
	}
	s.Conditions = conditions

	groups := query.ArrayStr("groups")
	if len(groups) > 0 {
		s.GroupBy(groups...)
	}

	havings := query.ArrayJson("havings")
	havingConditions, err := et.ToConditions(havings)
	if err != nil {
		return s, err
	}
	s.Havings = havingConditions

	limit := query.ValInt(s.MaxRows, "limit")
	s.setLimit(limit)

	page := query.ValInt(0, "page")
	s.setPage(page)

	orders := query.ArrayJson("orders")
	for _, order := range orders {
		for k := range order {
			v := order.ValBool(true, k)
			s.OrderBy(k, v)
		}
	}

	return s, nil
}
