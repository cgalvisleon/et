package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Table    string `json:"table"`
	As       string `json:"as"`
}

/**
* getFrom
* @param model *Model, as string
* @return *From
**/
func getFrom(model *Model, as string) *From {
	if as == "" {
		as = model.Table
	}
	return &From{
		Database: model.Database,
		Schema:   model.Schema,
		Name:     model.Name,
		Table:    model.Table,
		As:       as,
	}
}

type Field struct {
	Name string `json:"name"`
	As   string `json:"as"`
	From *From  `json:"from"`
}

type JoinType string

const (
	INNER_JOIN JoinType = "inner"
	LEFT_JOIN  JoinType = "left"
	RIGHT_JOIN JoinType = "right"
	FULL_JOIN  JoinType = "full"
)

type Join struct {
	Type      JoinType        `json:"type"`
	To        *From           `json:"to"`
	Condition []*et.Condition `json:"condition"`
	query     *Query          `json:"-"`
}

/**
* newJoin
* @param query *Query, typ JoinType, to *From, condition *et.Condition
* @return *Join
**/
func newJoin(query *Query, typ JoinType, to *From, condition *et.Condition) *Join {
	return &Join{
		Type:      typ,
		To:        to,
		Condition: []*et.Condition{condition},
		query:     query,
	}
}

type QuerySection int

const (
	whereSection QuerySection = iota
	joinSection
	havingSection
)

type Query struct {
	Froms      []*From         `json:"froms"`
	Joins      []*Join         `json:"joins"`
	Selects    []string        `json:"selects"`
	Conditions []*et.Condition `json:"conditions"`
	Hiddens    []string        `json:"hidden"`
	GroupsBy   []string        `json:"group_by"`
	OrdersBy   []*Index        `json:"order_by"`
	Havings    []*et.Condition `json:"havings"`
	Offset     int             `json:"offset"`
	Rows       int             `json:"rows"`
	section    QuerySection    `json:"-"`
	maxRows    int             `json:"-"`
	db         *DB             `json:"-"`
	isDebug    bool            `json:"-"`
	isTest     bool            `json:"-"`
}

/**
* newQuery
* @param model *Model, as ...string
* @return *Query
**/
func newQuery(model *Model, as ...string) *Query {
	if len(as) == 0 {
		as = []string{model.Table}
	}
	result := &Query{
		Froms:      make([]*From, 0),
		Joins:      make([]*Join, 0),
		Selects:    make([]string, 0),
		Conditions: make([]*et.Condition, 0),
		Hiddens:    make([]string, 0),
		GroupsBy:   make([]string, 0),
		OrdersBy:   make([]*Index, 0),
		Havings:    make([]*et.Condition, 0),
		section:    whereSection,
		maxRows:    model.db.RecordLimit,
		db:         model.db,
	}
	result.addFrom(model, as[0])
	return result
}

/**
* serialize
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
* ToJson
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
* Debug
* @return *Query
**/
func (s *Query) Debug() *Query {
	s.isDebug = true
	return s
}

/**
* Test
* @return *Query
**/
func (s *Query) Test() *Query {
	s.isTest = true
	return s
}

/**
* addFrom
* @param model *Model, as string
* @return *Query
**/
func (s *Query) addFrom(model *Model, as string) *Query {
	from := getFrom(model, as)
	s.Froms = append(s.Froms, from)
	return s
}

/**
* join
* @param model *Model, as string, tp JoinType, on *et.Condition
* @return *Query
**/
func (s *Query) join(model *Model, as string, tp JoinType, on *et.Condition) *Query {
	result := newJoin(s, tp, getFrom(model, as), on)
	s.Joins = append(s.Joins, result)
	s.section = joinSection
	return s
}

/**
* Join
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) Join(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, INNER_JOIN, on)
}

/**
* LeftJoin
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) LeftJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, LEFT_JOIN, on)
}

/**
* RightJoin
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) RightJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, RIGHT_JOIN, on)
}

/**
* FullJoin
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) FullJoin(model *Model, as string, on *et.Condition) *Query {
	return s.join(model, as, FULL_JOIN, on)
}

/**
* addCondition
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) addCondition(conds []*et.Condition) *Query {
	s.Conditions = append(s.Conditions, conds...)
	return s
}

/**
* Where
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Where(cond *et.Condition) *Query {
	s.Conditions = append(s.Conditions, cond)
	s.section = whereSection
	return s
}

/**
* And
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
* Or
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
* GroupBy
* @param fields ...string
* @return *Query
**/
func (s *Query) GroupBy(fields ...string) *Query {
	s.GroupsBy = append(s.GroupsBy, fields...)
	return s
}

/**
* Having
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Having(cond *et.Condition) *Query {
	s.Havings = append(s.Havings, cond)
	s.section = havingSection
	return s
}

/**
* Page
* @param page int
* @return *Query
**/
func (s *Query) Page(page int) *Query {
	s.Offset = (page - 1) * s.Rows
	return s
}

/**
* Limit
* @param rows int
* @return *Query
**/
func (s *Query) Limit(rows int) *Query {
	s.Rows = rows
	return s
}

/**
* AllTx
* @param tx *Tx
* @return (et.Items, error)
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
		logs.Debug("SQL:", sql)
	}

	if !s.isTest {
		result, err := s.db.sqlTx(tx, sql)
		if err != nil {
			return et.Items{}, err
		}
		return result, nil
	}

	return et.Items{}, nil
}

/**
* All
* @return (et.Items, error)
**/
func (s *Query) All() (et.Items, error) {
	result, err := s.AllTx(nil)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* OneTx
* @param tx *Tx
* @return (et.Item, error)
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
* One
* @return (et.Item, error)
**/
func (s *Query) One() (et.Item, error) {
	return s.OneTx(nil)
}
