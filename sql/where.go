package sql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Iterator interface {
	Next() (et.Json, bool)
	As() string
	Add(item et.Json)
	Data(int) et.Json
}

type Source struct {
	data []et.Json
	as   string
}

/**
* Next
* @return (et.Json, bool)
**/
func (s *Source) Next() (et.Json, bool) {
	if len(s.data) == 0 {
		return et.Json{}, false
	}
	item := s.data[0]
	s.data = s.data[1:]
	return item, true
}

/**
* As
* @return string
**/
func (s *Source) As() string {
	return s.as
}

/**
* Add
* @param item et.Json
**/
func (s *Source) Add(item et.Json) {
	s.data = append(s.data, item)
}

/**
* Data
* @param index int
* @return et.Json
**/
func (s *Source) Data(index int) et.Json {
	if index < 0 || index >= len(s.data) {
		return et.Json{}
	}
	return s.data[index]
}

type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
)

/**
* String
* @return string
**/
func (j JoinType) String() string {
	switch j {
	case InnerJoin:
		return "inner"
	case LeftJoin:
		return "left"
	case RightJoin:
		return "right"
	case FullJoin:
		return "full"
	default:
		return ""
	}
}

type Join struct {
	To   Iterator          `json:"to"`
	Keys map[string]string `json:"keys"`
	Type JoinType          `json:"type"`
}

/**
* Where
**/
type Where struct {
	From       Iterator        `json:"from"`
	Conditions []*Condition    `json:"conditions"`
	Selects    []string        `json:"selects"`
	Joins      []*Join         `json:"joins"`
	Hiddens    []string        `json:"hiddens"`
	OrderBy    map[string]bool `json:"order_by"`
	Offset     int             `json:"offset"`
	Limits     int             `json:"limits"`
	Workers    int             `json:"workers"`
	Result     []et.Json       `json:"result"`
	isDebug    bool            `json:"-"`
}

/**
* newWhere
* @param from []et.Json, as string
* @return *Where
**/
func newWhere(from []et.Json, as string) *Where {
	limitRows := envar.GetInt("LIMIT_ROWS", 1000)
	result := &Where{
		From: &Source{
			data: from,
			as:   as,
		},
		Conditions: make([]*Condition, 0),
		Selects:    make([]string, 0),
		Joins:      make([]*Join, 0),
		Hiddens:    make([]string, 0),
		OrderBy:    make(map[string]bool, 0),
		Offset:     0,
		Limits:     limitRows,
		Workers:    1,
		Result:     make([]et.Json, 0),
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Where
**/
func (s *Where) IsDebug() *Where {
	s.isDebug = true
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Where) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Add
* @param condition *Condition
* @return *Where
**/
func (s *Where) Add(condition *Condition) *Where {
	if len(s.Conditions) > 0 && condition.Connector == NaC {
		condition.Connector = And
	}

	s.Conditions = append(s.Conditions, condition)
	return s
}

/**
* Where
* @param condition *Condition
* @return *Where
**/
func (s *Where) Where(condition *Condition) *Where {
	return s.Add(condition)
}

/**
* And
* @param condition *Condition
* @return *Where
**/
func (s *Where) And(condition *Condition) *Where {
	condition.Connector = And
	return s.Add(condition)
}

/**
* Or
* @param condition *Condition
* @return *Where
**/
func (s *Where) Or(condition *Condition) *Where {
	condition.Connector = Or
	return s.Add(condition)
}

/**
* Select
* @param fields ...string
* @return *Where
**/
func (s *Where) Select(fields ...string) *Where {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.Selects = append(s.Selects, field)
	}

	return s
}

/**
* join
* @param to []et.Json, as string, keys map[string]string, joinType JoinType
* @return *Where
**/
func (s *Where) join(to []et.Json, as string, keys map[string]string, joinType JoinType) *Where {
	if len(to) == 0 {
		return s
	}

	join := &Join{
		To: &Source{
			data: to,
			as:   as,
		},
		Keys: keys,
		Type: joinType,
	}
	s.Joins = append(s.Joins, join)

	return s
}

/**
* Join
* @param to []et.Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) Join(to []et.Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, InnerJoin)
}

/**
* LeftJoin
* @param to []et.Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) LeftJoin(to []et.Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, LeftJoin)
}

/**
* RightJoin
* @param to []et.Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) RightJoin(to []et.Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, RightJoin)
}

/**
* FullJoin
* @param to []et.Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) FullJoin(to []et.Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, FullJoin)
}

/**
* Hidden
* @param fields ...string
* @return *Where
**/
func (s *Where) Hidden(fields ...string) *Where {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.Hiddens = append(s.Hiddens, field)
	}

	return s
}

/**
* Asc
* @param field string
* @return *Where
**/
func (s *Where) Asc(field string) *Where {
	s.OrderBy[field] = true
	return s
}

/**
* Desc
* @param field string
* @return *Where
**/
func (s *Where) Desc(field string) *Where {
	s.OrderBy[field] = false
	return s
}

/**
* Order
* @param field string
* @return *Where
**/
func (s *Where) Order(field string, asc bool) *Where {
	if asc {
		return s.Asc(field)
	}
	return s.Desc(field)
}

/**
* Limit
* @param page int, rows int
* @return *Where
**/
func (s *Where) Limit(page int, rows int) *Where {
	offset := (page - 1) * rows
	s.Limits = rows
	s.Offset = offset
	return s
}

/**
* AdddResult
* @return next bool
**/
func (s *Where) AdddResult(item et.Json) (next bool) {
	if len(s.Selects) == 0 {
		item = hidden(s.Hiddens, item)
		s.Result = append(s.Result, item)
	} else {
		item = hidden(s.Hiddens, item)
		item = selects(s.Selects, item)
		items := []et.Json{}
		for key, val := range item {
			vals := item.ArrayJson(key)
			if len(vals) == 0 {
				items = MergeToKeyValue(items, key, val)
			} else if len(items) == 0 {
				items = vals
			} else {
				items = MergeToMap(items, vals)
			}
		}

		s.Result = append(s.Result, items...)
	}

	if s.Limits == 0 {
		next = true
	} else {
		next = len(s.Result) < s.Limits
	}

	return
}

/**
* Run
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Where) Run(tx *Tx) []et.Json {
	tx, _ = GetTx(tx)

	from := s.From
	if len(s.Joins) == 0 && s.From.As() != "" {
		from = &Source{
			data: []et.Json{},
			as:   s.From.As(),
		}

		for {
			item, ok := s.From.Next()
			if !ok {
				break
			}

			item = Prefixer(item, s.From.As())
			from.Add(item)
		}
	} else {
		for _, join := range s.Joins {
			from = Joingy(from, join.To, join.Keys, join.Type)
		}
	}

	for {
		item, ok := from.Next()
		if !ok {
			break
		}
		ok = Validate(item, s.Conditions)
		if !ok {
			continue
		}
		next := s.AdddResult(item)
		if !next {
			break
		}
	}

	return s.Result
}

/**
* One
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) One(tx *Tx, idx int) et.Json {
	rows := s.Run(tx)
	n := len(rows)
	if n == 0 {
		return et.Json{}
	}

	if idx < 0 {
		idx = n + idx
	}

	if idx >= n {
		return et.Json{}
	}

	return rows[idx]
}

/**
* First
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) First(tx *Tx) et.Json {
	return s.One(tx, 0)
}

/**
* Last
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) Last(tx *Tx) et.Json {
	return s.One(tx, -1)
}
