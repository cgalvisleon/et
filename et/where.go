package et

import (
	"encoding/json"
	"fmt"
	"maps"
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
	err = decodeJSON(bt, &result)
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

	s.Selects = append(s.Selects, fields...)

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

	s.Hiddens = append(s.Hiddens, fields...)

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
* AddResult: Adds item to the result; next is false when the limit is reached
* @param item Json
* @return next bool
**/
func (s *Wheres) AddResult(item Json) (next bool) {
	s.addItem(item)

	if s.Limits == 0 {
		next = true
	} else {
		next = len(s.Result) < s.Limits
	}

	return
}

/**
* AdddResult: Deprecated: use AddResult.
* @param item Json
* @return next bool
**/
func (s *Wheres) AdddResult(item Json) (next bool) {
	return s.AddResult(item)
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
		ok = EvaluateObject(item, s.Conditions)
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
			next := s.AddResult(item)
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

type Source struct {
	data      []Json `json:"-"`
	as        string `json:"-"`
	pos       int    `json:"-"`
	limitRows int    `json:"-"`
}

/**
* Next
* @return (Json, bool)
**/
func (s *Source) Next() (Json, bool) {
	if s.pos >= len(s.data) {
		return Json{}, false
	}
	item := s.data[s.pos]
	s.pos++
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
* @param item Json
**/
func (s *Source) Add(item Json) {
	s.data = append(s.data, item)
}

/**
* Data
* @param index int
* @return Json
**/
func (s *Source) Data(index int) Json {
	if index < 0 || index >= len(s.data) {
		return Json{}
	}
	return s.data[index]
}

/**
* LimitRows
* @return int
**/
func (s *Source) LimitRows() int {
	if s.limitRows == 0 {
		s.limitRows = 1000
	}
	return s.limitRows
}

/**
* From
* @param itmes []Json, as ...string
* @return *Where
**/
func From(items []Json, as ...string) *Wheres {
	asStr := "A"
	if len(as) == 1 {
		asStr = as[0]
	}

	return newWhere(&Source{
		data: items,
		as:   asStr,
	})
}

/**
* getField
* @param field string
* @return keys []string, as string
**/
func getField(field string) ([]string, string) {
	if strings.Contains(field, ":") {
		parts := strings.Split(field, ":")
		return strings.Split(parts[0], "->"), parts[1]
	}
	return strings.Split(field, "->"), field
}

/**
* selects
* @param fields []string, object Json
* @return Json
**/
func selects(fields []string, object Json) Json {
	result := Json{}
	for _, field := range fields {
		keys, as := getField(field)
		val := object.Get(keys...)
		if val != nil {
			result[as] = val
		}
	}

	return result
}

/**
* hidden
* @param fields []string, object Json
* @return Json
**/
func hidden(fields []string, object Json) Json {
	if len(fields) == 0 {
		return object
	}

	hide := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		hide[f] = struct{}{}
	}

	result := Json{}
	for key, value := range object {
		if _, ok := hide[key]; ok {
			continue
		}
		result[key] = value
	}

	return result
}

/**
* MergeToKeyValue
* @params left []Json, key string, value any
* @return []Json
**/
func MergeToKeyValue(left []Json, key string, value any) []Json {
	if len(left) == 0 {
		return []Json{
			{key: value},
		}
	}

	result := make([]Json, 0, len(left))
	for _, item := range left {
		item[key] = value
		result = append(result, item)
	}

	return result
}

/**
* MergeToMap
* @params left, right []Json
* @return []Json
**/
func MergeToMap(left, right []Json) []Json {
	result := make([]Json, 0, len(left)*len(right))
	for _, itemL := range left {
		for _, itemR := range right {
			merged := make(Json, len(itemR)+len(itemL))
			maps.Copy(merged, itemR)
			maps.Copy(merged, itemL)
			result = append(result, merged)
		}
	}

	return result
}

/**
* applyPrefix: Builds a new Json with lowercased keys and an already-lowercase prefix.
* @param item Json, prefix string
* @return Json
**/
func applyPrefix(item Json, prefix string) Json {
	result := Json{}
	for k, v := range item {
		k = strings.ToLower(k)
		if prefix != "" {
			result[prefix+"."+k] = v
		} else {
			result[k] = v
		}
	}
	return result
}

/**
* Prefixer
* @param item Json, as string
* @return Json
**/
func Prefixer(item Json, as string) Json {
	return applyPrefix(item, strings.ToLower(as))
}

/**
* Joingy
* @params left, right Iterator, keys map[string]string, joinType JoinType
* @return Iterator
**/
func Joingy(left, right Iterator, keys map[string]string, joinType JoinType) Iterator {
	result := []Json{}
	rightIndex := map[string][]int{}
	matchedRight := map[int]bool{}
	rightItems := []Json{}

	rightAs := strings.ToLower(right.As())
	leftAs := strings.ToLower(left.As())

	// RIGHT: prefix and index every item
	for {
		r, ok := right.Next()
		if !ok {
			break
		}
		r = applyPrefix(r, rightAs)
		i := len(rightItems)
		rightItems = append(rightItems, r)
		key := buildKey(r, keys, false)
		rightIndex[key] = append(rightIndex[key], i)
	}

	// LEFT: match each item
	for {
		l, ok := left.Next()
		if !ok {
			break
		}

		l = applyPrefix(l, leftAs)
		key := buildKey(l, keys, true)
		ridxs, ok := rightIndex[key]
		if ok {
			for _, ri := range ridxs {
				matchedRight[ri] = true
				result = append(result, merge(l, rightItems[ri]))
			}
		} else {
			if joinType == LeftJoin || joinType == FullJoin {
				result = append(result, merge(l, nil))
			}
		}
	}

	// RIGHT/FULL: unmatched right rows
	if joinType == RightJoin || joinType == FullJoin {
		for i, r := range rightItems {
			if !matchedRight[i] {
				result = append(result, merge(nil, r))
			}
		}
	}

	return &Source{
		data: result,
		as:   leftAs,
	}
}

/**
* merge
* @params left, right Json, as string
* @return Json
**/
func merge(left, right Json) Json {
	out := Json{}

	if left != nil {
		maps.Copy(out, left)
	}

	for k, v := range right {
		if _, exists := out[k]; exists {
			continue
		}

		out[k] = v
	}

	return out
}

/**
* buildKey
* @params j Json, keys map[string]string, fromLeft bool
* @return string
**/
func buildKey(j Json, keys map[string]string, fromLeft bool) string {
	lks := make([]string, 0, len(keys))
	for lk := range keys {
		lks = append(lks, lk)
	}
	sort.Strings(lks)

	var sb strings.Builder
	for _, lk := range lks {
		field := keys[lk]
		if fromLeft {
			field = lk
		}
		if v, ok := j[field]; ok {
			sb.WriteByte('|')
			fmt.Fprintf(&sb, "%v", v)
		} else {
			sb.WriteString("|NULL")
		}
	}
	return sb.String()
}
