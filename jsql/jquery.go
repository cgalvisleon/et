package jsql

import (
	"errors"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
)

var keywords = []string{
	"select",
	"from",
	"join",
	"left join",
	"left_join",
	"right join",
	"right_join",
	"full join",
	"full_join",
	"where",
	"and",
	"or",
	"group by",
	"group_by",
	"having",
	"order by",
	"order_by",
	"limit",
	"offset",
	"insert",
	"update",
	"delete",
	"upsert",
	"define",
}

var keywordsQuery = []string{
	"select",
	"from",
}

var keywordsCommand = []string{
	"insert",
	"update",
	"delete",
	"upsert",
}

var keywordsDDL = []string{
	"define",
}

/**
* queryJson: Executes a query.
* @param sql et.Json
* @return *Query, error
**/
func (s *DB) queryJson(sql et.Json) (et.Items, error) {
	return s.queryJsonTx(nil, sql)
}

/**
* queryJsonTx: Executes a query with a transaction.
* @param tx *Tx, sql ry.Json
* @return *Query, error
**/
func (s *DB) queryJsonTx(tx *Tx, sql et.Json) (et.Items, error) {
	for k := range sql {
		if isKeywordQuery(k) {
			result, err := s.parseQuery(tx, sql)
			if err != nil {
				return et.Items{}, err
			}

			return result, nil
		} else if isKeywordCommand(k) {
			result, err := s.parseCommand(tx, sql)
			if err != nil {
				return et.Items{}, err
			}

			return result, nil
		} else if isKeywordDDL(k) {
			result, err := s.parseDDL(tx, sql)
			if err != nil {
				return et.Items{}, err
			}

			return result, nil
		}
	}

	return et.Items{}, errors.New("invalid sql")
}

func isKeywordQuery(k string) bool {
	k = strings.ToLower(k)
	return slices.Contains(keywordsQuery, k)
}

func isKeywordCommand(k string) bool {
	k = strings.ToLower(k)
	return slices.Contains(keywordsCommand, k)
}

func isKeywordDDL(k string) bool {
	k = strings.ToLower(k)
	return slices.Contains(keywordsDDL, k)
}

func (s *DB) parseQuery(tx *Tx, sql et.Json) (et.Items, error) {
	return et.Items{}, nil
}

func (s *DB) parseCommand(tx *Tx, sql et.Json) (et.Items, error) {
	return et.Items{}, nil
}

func (s *DB) parseDDL(tx *Tx, sql et.Json) (et.Items, error) {
	return et.Items{}, nil
}

func (s *DB) parseSelect(sql et.Json) []string {
	return []string{}
}

func (s *DB) parseFrom(sql et.Json) []*From {
	return nil
}

func (s *DB) parseJoin(sql et.Json) []*Join {
	return nil
}

func (s *DB) parseLeftJoin(sql et.Json) []*Join {
	return nil
}

func (s *DB) parseRightJoin(sql et.Json) []*Join {
	return nil
}

func (s *DB) parseFullJoin(sql et.Json) []*Join {
	return nil
}

func (s *DB) parseWhere(sql et.Json) []*et.Condition {
	return nil
}

func (s *DB) parseAnd(sql et.Json) []*et.Condition {
	return nil
}

func (s *DB) parseOr(sql et.Json) []*et.Condition {
	return nil
}

func (s *DB) parseGroupBy(sql et.Json) []string {
	return nil
}

func (s *DB) parseHaving(sql et.Json) []*et.Condition {
	return nil
}

func (s *DB) parseOrderBy(sql et.Json) []*Index {
	return nil
}

func (s *DB) parseLimit(sql et.Json, query *Query) *Query {
	return query
}

func (s *DB) parseInsert(sql et.Json) *Command {
	return nil
}

func (s *DB) parseUpdate(sql et.Json) *Command {
	return nil
}

func (s *DB) parseDelete(sql et.Json) *Command {
	return nil
}

func (s *DB) parseUpsert(sql et.Json) *Command {
	return nil
}

func (s *DB) parseDefine(sql et.Json) Define {
	return Define{}
}
