package jsql

import (
	"encoding/json"
	"regexp"
	"strconv"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* F: Identifies a table source with its fully-qualified name and SQL alias.
**/
type F struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Table    string `json:"table"`
	As       string `json:"as"`
	Model    *Model `json:"-"`
}

/**
* getFrom: Builds a F descriptor from a model, using as as the SQL alias (defaults to table name).
* @param model *Model
* @param as string
* @return *F
**/
func getFrom(model *Model, as string) *F {
	if as == "" {
		as = model.Table
	}
	return &F{
		Database: model.Database,
		Schema:   model.Schema,
		Name:     model.Name,
		Table:    model.Table,
		As:       as,
		Model:    model,
	}
}

/**
* Field: Represents a SELECT list entry with an optional alias and source table reference.
**/
type Field struct {
	TypeColumn TypeColumn `json:"type_column"`
	TypeData   TypeData   `json:"type_data"`
	Name       string     `json:"name"`
	As         string     `json:"as"`
	From       *F         `json:"from"`
	Agg        string     `json:"agg"`
	Page       int        `json:"page"`
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
	To        *F              `json:"to"`
	Condition []*et.Condition `json:"condition"`
	query     *Query          `json:"-"`
}

/**
* newJoin: Constructs a Join entry linked to its parent query.
* @param query *Query
* @param typ JoinType
* @param to *F
* @param condition *et.Condition
* @return *Join
**/
func newJoin(query *Query, typ JoinType, to *F, condition *et.Condition) *Join {
	return &Join{
		Type:      typ,
		To:        to,
		Condition: []*et.Condition{condition},
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
	To     *F                `json:"to"`
	Keys   map[string]string `json:"keys"`
	Select []string          `json:"select"`
	Page   int               `json:"page"`
	Rows   int               `json:"rows"`
}

/**
* GetQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *QueryDetail) GetQuery(item et.Json) *Query {
	q := newQuery(s.To.Model, "A")
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

/**
* Query: Holds all clauses needed to build a SELECT statement.
**/
type Query struct {
	Froms          []*F                    `json:"froms"`
	Joins          []*Join                 `json:"joins"`
	Selects        []string                `json:"selects"`
	Conditions     []*et.Condition         `json:"conditions"`
	Hiddens        []string                `json:"hidden"`
	GroupsBy       []string                `json:"group_by"`
	OrdersBy       []*Index                `json:"order_by"`
	Havings        []*et.Condition         `json:"havings"`
	Offset         int                     `json:"offset"`
	Rows           int                     `json:"rows"`
	UseSourceField bool                    `json:"use_source_field"`
	Details        map[string]*QueryDetail `json:"details"`
	Rollups        map[string]*QueryDetail `json:"rollups"`
	Calcs          map[string]CalcFunction `json:"calcs"`
	IsExists       bool                    `json:"is_exists"`
	IsCount        bool                    `json:"is_count"`
	section        QuerySection            `json:"-"`
	maxRows        int                     `json:"-"`
	db             *DB                     `json:"-"`
	isDebug        bool                    `json:"-"`
	isTest         bool                    `json:"-"`
}

/**
* newQuery: Creates a Query with the model as the primary FROM source.
* @param model *Model
* @param as ...string
* @return *Query
**/
func newQuery(model *Model, as ...string) *Query {
	if len(as) == 0 {
		as = []string{model.Table}
	}
	result := &Query{
		Froms:      make([]*F, 0),
		Joins:      make([]*Join, 0),
		Selects:    make([]string, 0),
		Conditions: make([]*et.Condition, 0),
		Hiddens:    make([]string, 0),
		GroupsBy:   make([]string, 0),
		OrdersBy:   make([]*Index, 0),
		Havings:    make([]*et.Condition, 0),
		Details:    make(map[string]*QueryDetail, 0),
		Rollups:    make(map[string]*QueryDetail, 0),
		Calcs:      make(map[string]CalcFunction, 0),
		section:    whereSection,
		maxRows:    model.db.RecordLimit,
		db:         model.db,
		isDebug:    model.db.IsDebug,
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
* ToJson: Returns the query metadata as an et.Json map.
* @return et.Json
**/
func (s *Query) ToJson() et.Json {
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
* Debug: Enables SQL logging for this query and returns it for chaining.
* @return *Query
**/
func (s *Query) Debug() *Query {
	return s.setDebug(true)
}

/**
* Test: Enables test mode — SQL is generated but not executed.
* @return *Query
**/
func (s *Query) Test() *Query {
	s.isTest = true
	return s
}

/**
* GetField: Creates a Field from a Column, using the Column's name and attaching the provided F.
* @param field string
* @return (*Field, bool)
**/
func (s *Query) GetField(field string) (*Field, bool) {
	pattern1 := regexp.MustCompile(`^([A-Za-z0-9_]+)\.([A-Za-z0-9_>-]+):([A-Za-z0-9_]+)$`) // from.field:as
	pattern2 := regexp.MustCompile(`^([A-Za-z0-9_]+)\.([A-Za-z0-9_>-]+)$`)                 // from.field
	pattern3 := regexp.MustCompile(`^([A-Za-z0-9_>-]+):([A-Za-z0-9_]+)$`)                  // field:as
	pattern4 := regexp.MustCompile(`^([A-Za-z0-9_>-]+)$`)                                  // field
	pattern5 := regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\):([A-Za-z0-9_]+)$`)            // agg(field):as
	pattern6 := regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\)`)                             // agg(field)
	pattern7 := regexp.MustCompile(`^([^|]+)\|page:(\d+)$`)                                // field|page:1

	getForm := func(name string) *F {
		if len(s.Froms) == 0 {
			return nil
		}
		if name == "" {
			return s.Froms[0]
		}
		for _, from := range s.Froms {
			if from.Name == name {
				return from
			} else if from.As == name {
				return from
			}
		}
		return nil
	}

	parseInt := func(s string, defaultValue int) int {
		if s == "" {
			return defaultValue
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return defaultValue
		}
		return i
	}

	page := 1
	if pattern7.MatchString(field) {
		matches := pattern7.FindStringSubmatch(field)
		if len(matches) == 3 {
			field = matches[1]
			page = parseInt(matches[2], 1)
			return nil, false
		}
	}

	if pattern1.MatchString(field) {
		matches := pattern1.FindStringSubmatch(field)
		if len(matches) == 4 {
			fromName := matches[1]
			columnName := matches[2]
			as := matches[3]
			from := getForm(fromName)
			if from == nil {
				return nil, false
			}
			col, ok := from.Model.GetColumn(columnName)
			if !ok {
				return nil, false
			}
			return &Field{
				TypeColumn: col.TypeColumn,
				TypeData:   col.TypeData,
				Name:       columnName,
				As:         as,
				From:       from,
				Page:       page,
			}, true
		}
	} else if pattern2.MatchString(field) {
		matches := pattern2.FindStringSubmatch(field)
		if len(matches) == 3 {
			fromName := matches[1]
			columnName := matches[2]
			from := getForm(fromName)
			if from == nil {
				return nil, false
			}
			col, ok := from.Model.GetColumn(columnName)
			if !ok {
				return nil, false
			}
			return &Field{
				TypeColumn: col.TypeColumn,
				TypeData:   col.TypeData,
				Name:       columnName,
				As:         columnName,
				From:       from,
				Page:       page,
			}, true
		}
	} else if pattern3.MatchString(field) {
		matches := pattern3.FindStringSubmatch(field)
		if len(matches) == 3 {
			columnName := matches[1]
			as := matches[2]
			from := getForm("")
			if from == nil {
				return nil, false
			}
			col, ok := from.Model.GetColumn(columnName)
			if !ok {
				return nil, false
			}
			return &Field{
				TypeColumn: col.TypeColumn,
				TypeData:   col.TypeData,
				Name:       columnName,
				As:         as,
				From:       from,
				Page:       page,
			}, true
		}
	} else if pattern4.MatchString(field) {
		matches := pattern4.FindStringSubmatch(field)
		if len(matches) == 2 {
			columnName := matches[1]
			from := getForm("")
			if from == nil {
				return nil, false
			}
			col, ok := from.Model.GetColumn(columnName)
			if !ok {
				return nil, false
			}
			return &Field{
				TypeColumn: col.TypeColumn,
				TypeData:   col.TypeData,
				Name:       columnName,
				As:         columnName,
				From:       from,
				Page:       page,
			}, true
		}
	} else if pattern5.MatchString(field) {
		matches := pattern5.FindStringSubmatch(field)
		if len(matches) == 4 {
			agg := matches[1]
			columnName := matches[2]
			as := matches[3]
			result, ok := s.GetField(columnName)
			if !ok {
				return nil, false
			}
			result.As = as
			result.Agg = agg
			return result, true
		}
	} else if pattern6.MatchString(field) {
		matches := pattern6.FindStringSubmatch(field)
		if len(matches) == 3 {
			agg := matches[1]
			columnName := matches[2]
			as := matches[1]
			result, ok := s.GetField(columnName)
			if !ok {
				return nil, false
			}
			result.As = as
			result.Agg = agg
			return result, true
		}
	}

	return nil, false
}

/**
* addFrom: Appends a FROM entry for the given model with the specified alias.
* @param model *Model
* @param as string
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
* @param model *Model
* @param as string
* @param tp JoinType
* @param on *et.Condition
* @return *Query
**/
func (s *Query) join(model *Model, as string, tp JoinType, on *et.Condition) *Query {
	result := newJoin(s, tp, getFrom(model, as), on)
	s.Joins = append(s.Joins, result)
	s.section = joinSection
	return s
}

/**
* Join: Appends an INNER JOIN clause.
* @param model *Model
* @param as string
* @param on *et.Condition
* @return *Query
**/
func (s *Query) Join(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, INNER_JOIN, on)
}

/**
* LeftJoin: Appends a LEFT JOIN clause.
* @param model *Model
* @param as string
* @param on *et.Condition
* @return *Query
**/
func (s *Query) LeftJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, LEFT_JOIN, on)
}

/**
* RightJoin: Appends a RIGHT JOIN clause.
* @param model *Model
* @param as string
* @param on *et.Condition
* @return *Query
**/
func (s *Query) RightJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, RIGHT_JOIN, on)
}

/**
* FullJoin: Appends a FULL JOIN clause.
* @param model *Model
* @param as string
* @param on *et.Condition
* @return *Query
**/
func (s *Query) FullJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, FULL_JOIN, on)
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
* Select: Appends fields to the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Select(fields ...string) *Query {
	s.Selects = append(s.Selects, fields...)
	return s
}

/**
* Hidden: Appends fields to the HIDDEN clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Hidden(fields ...string) *Query {
	s.Hiddens = append(s.Hiddens, fields...)
	return s
}

/**
* Where: Appends a condition to the WHERE clause and sets the active section to where.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Where(cond *et.Condition) *Query {
	if len(s.Conditions) > 0 {
		return s.And(cond)
	}
	s.Conditions = append(s.Conditions, cond)
	s.section = whereSection
	return s
}

/**
* And: Appends an AND condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) And(cond *et.Condition) *Query {
	cond.Connector = et.And
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
* Or: Appends an OR condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Or(cond *et.Condition) *Query {
	cond.Connector = et.Or
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
* GroupBy: Adds one or more fields to the GROUP BY clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) GroupBy(fields ...string) *Query {
	s.GroupsBy = append(s.GroupsBy, fields...)
	return s
}

/**
* Having: Appends a condition to the HAVING clause and sets the active section to having.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Having(cond *et.Condition) *Query {
	s.Havings = append(s.Havings, cond)
	s.section = havingSection
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
* Page: Sets the result offset based on the 1-based page number and current Rows limit.
* @param page int
* @return *Query
**/
func (s *Query) Page(page int) *Query {
	return s.setPage(page)
}

/**
* OrderBy: Appends a field to the ORDER BY clause; sorted=true means ASC, false means DESC.
* @param field string
* @param sorted bool
* @return *Query
**/
func (s *Query) OrderBy(field string, sorted ...bool) *Query {
	sortedValue := true
	if len(sorted) > 0 {
		sortedValue = sorted[0]
	}
	s.OrdersBy = append(s.OrdersBy, &Index{Name: field, Sorted: sortedValue})
	return s
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
	return item
}

/**
* setRollup: Sets the rollup for the query.
* @param tx *Tx
* @param item et.Json
* @return et.Json
**/
func (s *Query) setRollup(tx *Tx, item et.Json) et.Json {
	for name, detail := range s.Rollups {
		qry := detail.GetQuery(item)
		detailResult, err := qry.AllTx(tx)
		if err != nil {
			return item
		}
		if !detailResult.Ok {
			continue
		}
		if len(detail.Select) == 1 {
			att := detail.Select[0]
			val := detailResult.Get(0, att)
			item[att] = val
			continue
		}
		item[name] = detailResult.Result[0]
	}
	return item
}

/**
* setCalcs: Sets the calculations for the query.
* @param tx *Tx, item et.Json
* @return et.Json
**/
func (s *Query) setCalcs(tx *Tx, item et.Json) et.Json {
	for _, calc := range s.Calcs {
		calc(tx, item)
	}
	return item
}

/**
* AllTx: Generates and executes a SELECT query inside the given transaction.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Query) AllTx(tx *Tx) (et.Items, error) {
	if s.Rows == 0 {
		s.Rows = s.maxRows
	}

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
		s.setCalcs(tx, item)
		result.Result[i] = item
	}

	return result, nil
}

/**
* All: Generates and executes a SELECT query without an explicit transaction.
* @return et.Items, error
**/
func (s *Query) All() (et.Items, error) {
	result, err := s.AllTx(nil)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* OneTx: Executes the query limited to one row inside the given transaction.
* @param tx *Tx
* @return et.Item, error
**/
func (s *Query) OneTx(tx *Tx) (et.Item, error) {
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
* One: Executes the query limited to one row without an explicit transaction.
* @return et.Item, error
**/
func (s *Query) One() (et.Item, error) {
	return s.OneTx(nil)
}

/**
* Limit: Sets the maximum number of rows to return.
* @param tx *Tx, page int, rows int
* @return et.Items, error
**/
func (s *Query) LimitTx(tx *Tx, page, rows int) (et.Items, error) {
	if rows > s.maxRows {
		rows = s.maxRows
	}
	s.Rows = rows
	s.setPage(page)
	return s.AllTx(tx)
}

/**
* Limit: Sets the maximum number of rows to return.
* @param page int, rows int
* @return et.Items, error
**/
func (s *Query) Limit(page, rows int) (et.Items, error) {
	return s.LimitTx(nil, page, rows)
}

/**
* PrimaryModel: Returns the Model for the primary FROM source, or nil if not found.
* @return *Model
**/
func (s *Query) ExistsTx(tx *Tx) (bool, error) {
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
* Exists: Checks if any rows match the query conditions.
* @return bool, error
**/
func (s *Query) Exists() (bool, error) {
	return s.ExistsTx(nil)
}

/**
* CountTx: Executes the query and returns the count of matching rows within the given transaction.
* @param tx *Tx
* @return int, error
**/
func (s *Query) CountTx(tx *Tx) (int, error) {
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
* Count: Executes the query and returns the count of matching rows.
* @return int, error
**/
func (s *Query) Count() (int, error) {
	return s.CountTx(nil)
}
