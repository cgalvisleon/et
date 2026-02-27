package sql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

/**
* Where
**/
type Where struct {
	From       []et.Json       `json:"from"`
	Conditions []*Condition    `json:"conditions"`
	Selects    []string        `json:"selects"`
	Joins      []string        `json:"joins"`
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
* @param owner *From
* @return *Where
**/
func newWhere(from []et.Json) *Where {
	limitRows := envar.GetInt("LIMIT_ROWS", 1000)
	result := &Where{
		From:       from,
		Conditions: make([]*Condition, 0),
		Selects:    make([]string, 0),
		Joins:      make([]string, 0),
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
* Join
* @param fields ...string
* @return *Where
**/
func (s *Where) Join(fields ...string) *Where {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.Joins = append(s.Joins, field)
	}

	return s
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
	if len(s.Joins) > 0 {
		item = selects(s.Joins, item)
		items := []et.Json{}
		for key, val := range item {
			vals := item.ArrayJson(key)
			if len(vals) == 0 {
				items = JoinToKeyValue(items, key, val)
			} else if len(items) == 0 {
				items = vals
			} else {
				items = JoinToMap(items, vals)
			}
		}

		s.Result = append(s.Result, items...)
	} else if len(s.Selects) == 0 {
		item = hidden(s.Hiddens, item)
		s.Result = append(s.Result, item)
	} else {
		item = selects(s.Selects, item)
		item = hidden(s.Hiddens, item)
		s.Result = append(s.Result, item)
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

	for _, item := range s.From {
		ok := Validate(item, s.Conditions)
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
