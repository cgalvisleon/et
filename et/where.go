package et

import (
	"encoding/json"
	"sort"
	"strings"
)

type Iterator interface {
	Next() (Json, bool)
	As() string
	Add(item Json)
	LimitRows() int
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

type OrderField struct {
	Field string
	Asc   bool
}

/**
* Where
**/
type Wheres struct {
	From       Iterator     `json:"from"`
	Conditions []*Condition `json:"conditions"`
	Selects    []string     `json:"selects"`
	Joins      []*Join      `json:"joins"`
	Hiddens    []string     `json:"hiddens"`
	OrderBy    []OrderField `json:"order_by"`
	Offset     int          `json:"offset"`
	Limits     int          `json:"limits"`
	Workers    int          `json:"workers"`
	Result     []Json       `json:"result"`
	isDebug    bool         `json:"-"`
}

/**
* newWhere
* @param from []Json, as string
* @return *Wheres
**/
func newWhere(from Iterator) *Wheres {
	result := &Wheres{
		From:       from,
		Conditions: make([]*Condition, 0, 4),
		Selects:    make([]string, 0, 4),
		Joins:      make([]*Join, 0, 2),
		Hiddens:    make([]string, 0, 4),
		OrderBy:    make([]OrderField, 0, 2),
		Offset:     0,
		Limits:     from.LimitRows(),
		Workers:    1,
		Result:     make([]Json, 0, from.LimitRows()),
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Wheres
**/
func (s *Wheres) IsDebug() *Wheres {
	s.isDebug = true
	return s
}

/**
* ToJson
* @return Json
**/
func (s *Wheres) ToJson() (Json, error) {
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
* @return *Wheres
**/
func (s *Wheres) Add(condition *Condition) *Wheres {
	if len(s.Conditions) > 0 && condition.Connector == NaC {
		condition.Connector = AND
	}

	s.Conditions = append(s.Conditions, condition)
	return s
}

/**
* Where
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) Where(condition *Condition) *Wheres {
	return s.Add(condition)
}

/**
* And
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) And(condition *Condition) *Wheres {
	condition.Connector = AND
	return s.Add(condition)
}

/**
* Or
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) Or(condition *Condition) *Wheres {
	condition.Connector = OR
	return s.Add(condition)
}

/**
* Select
* @param fields ...string
* @return *Wheres
**/
func (s *Wheres) Select(fields ...string) *Wheres {
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
* @return *Wheres
**/
func (s *Wheres) join(to []Json, as string, keys map[string]string, joinType JoinType) *Wheres {
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
* @return *Wheres
**/
func (s *Wheres) Join(to []Json, as string, keys map[string]string) *Wheres {
	return s.join(to, as, keys, InnerJoin)
}

/**
* LeftJoin
* @param to []Json, as string, keys map[string]string
* @return *Wheres
**/
func (s *Wheres) LeftJoin(to []Json, as string, keys map[string]string) *Wheres {
	return s.join(to, as, keys, LeftJoin)
}

/**
* RightJoin
* @param to []Json, as string, keys map[string]string
* @return *Wheres
**/
func (s *Wheres) RightJoin(to []Json, as string, keys map[string]string) *Wheres {
	return s.join(to, as, keys, RightJoin)
}

/**
* FullJoin
* @param to []Json, as string, keys map[string]string
* @return *Wheres
**/
func (s *Wheres) FullJoin(to []Json, as string, keys map[string]string) *Wheres {
	return s.join(to, as, keys, FullJoin)
}

/**
* Hidden
* @param fields ...string
* @return *Wheres
**/
func (s *Wheres) Hidden(fields ...string) *Wheres {
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
* @return *Wheres
**/
func (s *Wheres) Asc(field string) *Wheres {
	s.OrderBy = append(s.OrderBy, OrderField{Field: field, Asc: true})
	return s
}

/**
* Desc
* @param field string
* @return *Wheres
**/
func (s *Wheres) Desc(field string) *Wheres {
	s.OrderBy = append(s.OrderBy, OrderField{Field: field, Asc: false})
	return s
}

/**
* Order
* @param field string
* @return *Wheres
**/
func (s *Wheres) Order(field string, asc bool) *Wheres {
	if asc {
		return s.Asc(field)
	}
	return s.Desc(field)
}

/**
* Limit
* @param page int, rows int
* @return *Wheres
**/
func (s *Wheres) Limit(page int, rows int) *Wheres {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * rows
	s.Limits = rows
	s.Offset = offset
	return s
}

/**
* addItem: Applies select/hidden transforms and appends item to Result without limit check.
* @param item Json
**/
func (s *Wheres) addItem(item Json) {
	if len(s.Selects) == 0 {
		item = hidden(s.Hiddens, item)
		s.Result = append(s.Result, item)
	} else {
		item = hidden(s.Hiddens, item)
		item = selects(s.Selects, item)
		items := make([]Json, 0, len(item))
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
}

/**
* AdddResult
* @param item Json
* @return next bool
**/
func (s *Wheres) AdddResult(item Json) (next bool) {
	s.addItem(item)

	if s.Limits == 0 {
		next = true
	} else {
		next = len(s.Result) < s.Limits
	}

	return
}

/**
* sortResult: Sorts Result in-place using OrderBy fields.
**/
func (s *Wheres) sortResult() {
	if len(s.OrderBy) == 0 || len(s.Result) == 0 {
		return
	}

	sort.SliceStable(s.Result, func(i, j int) bool {
		for _, of := range s.OrderBy {
			a := s.Result[i][of.Field]
			b := s.Result[j][of.Field]
			cmp, ok := compareAnyOrdered(a, b)
			if !ok || cmp == 0 {
				continue
			}
			if of.Asc {
				return cmp < 0
			}
			return cmp > 0
		}
		return false
	})
}

/**
* All
* @return []Json
**/
func (s *Wheres) All() []Json {
	from := s.From
	if len(s.Joins) == 0 && s.From.As() != "" {
		fromAs := strings.ToLower(s.From.As())
		from = &Source{
			data: []Json{},
			as:   fromAs,
		}

		for {
			item, ok := s.From.Next()
			if !ok {
				break
			}

			item = applyPrefix(item, fromAs)
			from.Add(item)
		}
	} else {
		for _, join := range s.Joins {
			from = Joingy(from, join.To, join.Keys, join.Type)
		}
	}

	hasOrder := len(s.OrderBy) > 0
	skipped := 0

	for {
		item, ok := from.Next()
		if !ok {
			break
		}
		ok = evaluateObject(item, s.Conditions)
		if !ok {
			continue
		}
		if hasOrder {
			// Collect all matching items before sorting; offset+limit applied after.
			s.addItem(item)
		} else {
			if skipped < s.Offset {
				skipped++
				continue
			}
			next := s.AdddResult(item)
			if !next {
				break
			}
		}
	}

	if hasOrder {
		s.sortResult()
		start := s.Offset
		if start < 0 {
			start = 0
		}
		if start > len(s.Result) {
			start = len(s.Result)
		}
		s.Result = s.Result[start:]
		if s.Limits > 0 && len(s.Result) > s.Limits {
			s.Result = s.Result[:s.Limits]
		}
	}

	return s.Result
}

/**
* evaluateObject
* @param item Json, conditions []*Condition
* @return bool
**/
func evaluateObject(item Json, conditions []*Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	result := conditions[0].ApplyToObject(item)
	for _, cond := range conditions[1:] {
		ok := cond.ApplyToObject(item)
		if cond.Connector == AND {
			result = result && ok
		} else if cond.Connector == OR {
			result = result || ok
		}
	}

	return result
}

/**
* One
* @param idx int
* @return Json
**/
func (s *Wheres) One(idx int) Json {
	rows := s.All()
	if len(rows) == 0 {
		return Json{}
	}

	n := len(rows)
	if idx < 0 {
		idx = n + idx
	}

	if idx < 0 || idx >= n {
		return Json{}
	}

	return rows[idx]
}

/**
* First
* @return Json
**/
func (s *Wheres) First() Json {
	return s.One(0)
}

/**
* Last
* @return Json
**/
func (s *Wheres) Last() Json {
	return s.One(-1)
}
