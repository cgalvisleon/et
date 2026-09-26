package jsql

import (
	"fmt"
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

/**
* Query: Executes a query.
* @param sql et.Json
* @return et.Items, error
**/
func (s *DB) Query(sql et.Json) (et.Items, error) {
	return s.QueryTx(nil, sql)
}

/**
* QueryTx: Executes a query with a transaction.
* @param tx *Tx, sql ry.Json
* @return et.Items, error
**/
func (s *DB) QueryTx(tx *Tx, sql et.Json) (et.Items, error) {
	sqlStr := ""
	for k := range sql {
		if isKeyword(k) {
			v := sql[k]
			sqlStr = parseSql(k, v, sqlStr)
		}
	}

	if sqlStr == "" {
		return et.Items{}, fmt.Errorf("invalid sql")
	}

	result, err := s.SqlTx(tx, sqlStr)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

func isKeyword(k string) bool {
	k = strings.ToLower(k)
	return slices.Contains(keywords, k)
}

func parseSql(k string, v any, sql string) string {
	k = strings.ToLower(k)
	switch k {
	case "select":
		return parseSelect(v, sql)
	case "from":
		return parseFrom(v, sql)
	case "join":
		return parseJoin(v, sql)
	case "left join":
		return parseLeftJoin(v, sql)
	case "left_join":
		return parseLeftJoin(v, sql)
	case "right join":
		return parseRightJoin(v, sql)
	case "right_join":
		return parseRightJoin(v, sql)
	case "full join":
		return parseFullJoin(v, sql)
	case "full_join":
		return parseFullJoin(v, sql)
	case "where":
		return parseWhere(v, sql)
	case "and":
		return parseAnd(v, sql)
	case "or":
		return parseOr(v, sql)
	case "group by":
		return parseGroupBy(v, sql)
	case "group_by":
		return parseGroupBy(v, sql)
	case "having":
		return parseHaving(v, sql)
	case "order by":
		return parseOrderBy(v, sql)
	case "order_by":
		return parseOrderBy(v, sql)
	case "limit":
		return parseLimit(v, sql)
	case "offset":
		return parseOffset(v, sql)
	case "insert":
		return parseInsert(v, sql)
	case "update":
		return parseUpdate(v, sql)
	case "delete":
		return parseDelete(v, sql)
	case "upsert":
		return parseUpsert(v, sql)
	case "define":
		return parseDefine(v, sql)
	}
	return ""
}

func parseSelect(v any, sql string) string {
	return ""
}

func parseFrom(v any, sql string) string {
	return ""
}

func parseJoin(v any, sql string) string {
	return ""
}

func parseLeftJoin(v any, sql string) string {
	return ""
}

func parseRightJoin(v any, sql string) string {
	return ""
}

func parseFullJoin(v any, sql string) string {
	return ""
}

func parseWhere(v any, sql string) string {
	return ""
}

func parseAnd(v any, sql string) string {
	return ""
}

func parseOr(v any, sql string) string {
	return ""
}

func parseGroupBy(v any, sql string) string {
	return ""
}

func parseHaving(v any, sql string) string {
	return ""
}

func parseOrderBy(v any, sql string) string {
	return ""
}

func parseLimit(v any, sql string) string {
	return ""
}

func parseOffset(v any, sql string) string {
	return ""
}

func parseInsert(v any, sql string) string {
	return ""
}

func parseUpdate(v any, sql string) string {
	return ""
}

func parseDelete(v any, sql string) string {
	return ""
}

func parseUpsert(v any, sql string) string {
	return ""
}

func parseDefine(v any, sql string) string {
	return ""
}
