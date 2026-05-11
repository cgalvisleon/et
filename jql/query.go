package jql

import (
	"encoding/json"
	"regexp"

	"github.com/cgalvisleon/et/et"
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

type Query struct {
	Froms      []*From         `json:"froms"`
	Selects    []string        `json:"selects"`
	Conditions []*et.Condition `json:"conditions"`
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
		Selects:    make([]string, 0),
		Conditions: make([]*et.Condition, 0),
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
* addFrom
* @param model *Model, as string
* @return *Query
**/
func (q *Query) addFrom(model *Model, as string) *Query {
	from := getFrom(model, as)
	q.Froms = append(q.Froms, from)
	return q
}

/**
* GetField
* @param name string
* @return *Field
**/
func (s *Query) GetField(name string) *Field {
	pattern1 := regexp.MustCompile(`^([A-Za-z0-9>]+):([A-Za-z0-9]+)$`) // name:as
	pattern2 := regexp.MustCompile(`^([A-Za-z0-9>]+)$`)                // name

	if pattern1.MatchString(name) {
		matches := pattern1.FindStringSubmatch(name)
		if len(matches) == 3 {
			name = matches[1]
			as := matches[2]
			column := s.FindColumn(name)
			if column != nil {
				result := column.Field()
				result.As = as
				return result
			}
		}
	} else if pattern2.MatchString(name) {
		column := s.FindColumn(name)
		if column != nil {
			result := column.Field()
			return result
		}
	}

	return nil
}

/**
* Where
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Where(cond *et.Condition) *Query {
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* And
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) And(cond *et.Condition) *Query {
	cond.Connector = et.And
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* Or
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Or(cond *et.Condition) *Query {
	cond.Connector = et.Or
	s.Conditions = append(s.Conditions, cond)
	return s
}
