package et

import (
	"encoding/json"

	"github.com/cgalvisleon/et/envar"
)

type Iterator interface {
	Next() (Json, bool)
	As() string
	Add(item Json)
	Data(int) Json
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
	Result     []Json          `json:"result"`
	isDebug    bool            `json:"-"`
}

/**
* newWhere
* @param from []Json, as string
* @return *Where
**/
func newWhere(from Iterator) *Where {
	limitRows := envar.GetInt("LIMIT_ROWS", 1000)
	result := &Where{
		From:       from,
		Conditions: make([]*Condition, 0),
		Selects:    make([]string, 0),
		Joins:      make([]*Join, 0),
		Hiddens:    make([]string, 0),
		OrderBy:    make(map[string]bool, 0),
		Offset:     0,
		Limits:     limitRows,
		Workers:    1,
		Result:     make([]Json, 0),
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
* @return Json
**/
func (s *Where) ToJson() (Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result Json
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
* @param to []Json, as string, keys map[string]string, joinType JoinType
* @return *Where
**/
func (s *Where) join(to []Json, as string, keys map[string]string, joinType JoinType) *Where {
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
* @param to []Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) Join(to []Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, InnerJoin)
}

/**
* LeftJoin
* @param to []Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) LeftJoin(to []Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, LeftJoin)
}

/**
* RightJoin
* @param to []Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) RightJoin(to []Json, as string, keys map[string]string) *Where {
	return s.join(to, as, keys, RightJoin)
}

/**
* FullJoin
* @param to []Json, as string, keys map[string]string
* @return *Where
**/
func (s *Where) FullJoin(to []Json, as string, keys map[string]string) *Where {
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
func (s *Where) AdddResult(item Json) (next bool) {
	if len(s.Selects) == 0 {
		item = hidden(s.Hiddens, item)
		s.Result = append(s.Result, item)
	} else {
		item = hidden(s.Hiddens, item)
		item = selects(s.Selects, item)
		items := []Json{}
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
* All
* @return ([]Json, bool)
**/
func (s *Where) All() ([]Json, bool) {
	from := s.From
	if len(s.Joins) == 0 && s.From.As() != "" {
		from = &Source{
			data: []Json{},
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
		ok = Evaluate(item, s.Conditions)
		if !ok {
			continue
		}
		next := s.AdddResult(item)
		if !next {
			break
		}
	}

	return s.Result, len(s.Result) > 0
}

/**
* One
* @param idx int
* @return Json, bool
**/
func (s *Where) One(idx int) (Json, bool) {
	rows, ok := s.All()
	if !ok {
		return Json{}, false
	}

	n := len(rows)
	if idx < 0 {
		idx = n + idx
	}

	if idx >= n {
		return Json{}, false
	}

	return rows[idx], true
}

/**
* First
* @return Json, bool
**/
func (s *Where) First() (Json, bool) {
	return s.One(0)
}

/**
* Last
* @return Json, bool
**/
func (s *Where) Last() (Json, bool) {
	return s.One(-1)
}
