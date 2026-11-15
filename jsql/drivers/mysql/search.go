package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

/**
* mySearchOf: The fields of a search condition and whether it goes with MATCH: a search of the model whose fields are
* all text columns (the ones a FULLTEXT index takes); a field alone or a search with attributes goes with LIKE.
* @param getField func(string) (*jsql.Field, bool), field string
* @return []*jsql.Field, *jsql.TextSearch, bool, bool (fields, search, match, ok)
**/
func mySearchOf(getField func(string) (*jsql.Field, bool), field string) ([]*jsql.Field, *jsql.TextSearch, bool, bool) {
	fields, search, ok := jsql.SearchFields(getField, field)
	if !ok || len(fields) == 0 {
		return nil, nil, false, false
	}
	match := search != nil && !slices.ContainsFunc(fields, func(fld *jsql.Field) bool { return !myFulltextColumn(fld) })
	return fields, search, match, true
}

/**
* myFulltextColumn: Whether a field can be in a FULLTEXT index: a column of text (VARCHAR or LONGTEXT).
* @param fld *jsql.Field
* @return bool
**/
func myFulltextColumn(fld *jsql.Field) bool {
	if fld.TypeColumn != jsql.COLUMN {
		return false
	}
	switch fld.TypeData {
	case et.TEXT, et.KEY, et.MEMO:
		return true
	}
	return false
}

/**
* myBooleanText: A search text (et.ToSearch) in the boolean mode of MATCH: every term required (+), the excluded with
* -, a phrase between quotes, and several groups as optional (+a +b) (+c). The operators of boolean mode are removed
* from the words; empty when nothing is left.
* @param text string
* @return string
**/
func myBooleanText(text string) string {
	clean := strings.NewReplacer("+", " ", "-", " ", ">", " ", "<", " ", "(", " ", ")", " ", "~", " ", "*", " ", `"`, " ", "@", " ")
	groups := []string{}
	for _, group := range et.ToSearch(text) {
		terms := []string{}
		for _, term := range group {
			value := strings.Join(strings.Fields(clean.Replace(term.Text)), " ")
			if value == "" {
				continue
			}
			op := "+"
			if term.Exclude {
				op = "-"
			}
			if term.Phrase || strings.Contains(value, " ") {
				value = `"` + value + `"`
			}
			terms = append(terms, op+value)
		}
		if len(terms) > 0 {
			groups = append(groups, strings.Join(terms, " "))
		}
	}
	if len(groups) > 1 {
		for i, group := range groups {
			groups[i] = "(" + group + ")"
		}
	}
	return strings.Join(groups, " ")
}

/**
* myMatch: MATCH(a, b) AGAINST ('…' IN BOOLEAN MODE) over the columns of a search; the text is always quoted.
* @param fields []*jsql.Field, text string
* @return string, bool (false when the text has no term)
**/
func myMatch(fields []*jsql.Field, text string) (string, bool) {
	against := myBooleanText(text)
	if against == "" {
		return "", false
	}
	cols := make([]string, 0, len(fields))
	for _, fld := range fields {
		cols = append(cols, myColumnRef(myAlias(fld.From), fld.Name))
	}
	return fmt.Sprintf("MATCH(%s) AGAINST (%s IN BOOLEAN MODE)", strings.Join(cols, ", "), myQuoteText(against)), true
}

/**
* mySearchExpr: Renders a search condition: MATCH … AGAINST for a search over text columns (its FULLTEXT index), LIKE
* for the rest. A field that does not resolve, or a text without terms, never matches.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition
* @return string
**/
func mySearchExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition) string {
	fields, _, match, ok := mySearchOf(getField, cond.Field.String())
	text, _, isText := jsql.SearchOf(cond) // MATCH and LIKE do not change with the lang
	if !ok || !isText {
		return "1 = 0"
	}
	if match {
		expr, ok := myMatch(fields, text)
		if !ok {
			return "1 = 0"
		}
		return expr
	}
	exprs := make([]string, 0, len(fields))
	for _, fld := range fields {
		exprs = append(exprs, myScalarExpr(fld, et.TEXT))
	}
	return jsql.SearchLike(exprs, text, func(expr, pattern string) string {
		return fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", expr, myQuoteText(pattern))
	})
}

/**
* mySearchRank: The ORDER BY of a search field: its MATCH against the text of the search condition of the where;
* false without that condition or when the search goes with LIKE.
* @param query *jsql.Query, name string
* @return string, bool
**/
func mySearchRank(query *jsql.Query, name string) (string, bool) {
	cond, ok := query.SearchCondition(name)
	if !ok {
		return "", false
	}
	fields, _, match, ok := mySearchOf(query.GetField, cond.Field.String())
	text, _, isText := jsql.SearchOf(cond)
	if !ok || !match || !isText {
		return "", false
	}
	return myMatch(fields, text)
}

/**
* myFulltextName: The name of the FULLTEXT index of a search, with a hash of its columns (a search whose fields change
* gets a new index and ApplyIndexes drops the old one).
* @param model *jsql.Model, name string, cols []string
* @return string
**/
func myFulltextName(model *jsql.Model, name string, cols []string) string {
	h := fnv.New32a()
	h.Write([]byte(strings.Join(cols, ",")))
	return strings.Trim(myObjectName(model.Name, name, "fts", fmt.Sprintf("%08x", h.Sum32())), "`")
}

// myIndexLock: the named lock that serializes the index DDL of the services that start at the same time
const myIndexLock = "et.jsql.indexes"

// mySearchIndexName: the name of a search index jsql made: <model>_<search>_fts_<hash of its columns>
var mySearchIndexName = regexp.MustCompile(`_fts_[0-9a-f]{8}$`)

/**
* myIndex: A FULLTEXT index the model wants: its name and its ALTER TABLE.
**/
type myIndex struct {
	name   string
	create string
}

/**
* myFulltextIndexes: The FULLTEXT index of each search whose fields are all text columns (one per search: MATCH does
* not depend on the lang, so the langs of a search do not add indexes in MySQL).
* @param model *jsql.Model
* @return []myIndex
**/
func myFulltextIndexes(model *jsql.Model) []myIndex {
	names := make([]string, 0, len(model.Searches))
	for name := range model.Searches {
		names = append(names, name)
	}
	slices.Sort(names)

	result := []myIndex{}
	for _, name := range names {
		fields, _, match, ok := mySearchOf(model.GetField, name)
		if !ok || !match {
			continue
		}
		cols := make([]string, 0, len(fields))
		for _, fld := range fields {
			cols = append(cols, myIdent(fld.Name))
		}
		index := myFulltextName(model, name, cols)
		result = append(result, myIndex{
			name: index,
			// The least blocking form InnoDB has for a FULLTEXT: in place, reads go on, writes wait (LOCK=NONE is not
			// allowed); the first FULLTEXT of a table without FTS_DOC_ID rebuilds it
			create: fmt.Sprintf("ALTER TABLE %s ADD FULLTEXT INDEX `%s` (%s), ALGORITHM=INPLACE, LOCK=SHARED", ddlTable(model), index, strings.Join(cols, ", ")),
		})
	}
	return result
}

/**
* myQueryer: A *sql.DB or the *sql.Conn of the lock.
**/
type myQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

/**
* myIndexesOf: The indexes of the table of the model, by name, with whether each is a FULLTEXT one.
* @param ctx context.Context, db myQueryer, model *jsql.Model
* @return map[string]bool, error
**/
func myIndexesOf(ctx context.Context, db myQueryer, model *jsql.Model) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT DISTINCT index_name, index_type = 'FULLTEXT' FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ?", sanitizeIdent(model.Table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]bool{}
	for rows.Next() {
		var name string
		var fulltext bool
		if err := rows.Scan(&name, &fulltext); err != nil {
			return nil, err
		}
		result[name] = fulltext
	}
	return result, rows.Err()
}

/**
* myIndexPlan: The statements that take the table from its indexes to the FULLTEXT ones the model wants: the missing
* are created and a FULLTEXT index of jsql (its name with the hash) that no search wants is dropped; any other index is
* never touched.
* @param model *jsql.Model, want []myIndex, have map[string]bool
* @return []string
**/
func myIndexPlan(model *jsql.Model, want []myIndex, have map[string]bool) []string {
	result, wanted := []string{}, map[string]bool{}
	for _, idx := range want {
		wanted[idx.name] = true
		if _, ok := have[idx.name]; !ok {
			result = append(result, idx.create)
		}
	}
	orphans := []string{}
	for name, fulltext := range have {
		if fulltext && !wanted[name] && mySearchIndexName.MatchString(name) {
			orphans = append(orphans, name)
		}
	}
	slices.Sort(orphans)
	for _, name := range orphans {
		result = append(result, fmt.Sprintf("ALTER TABLE %s DROP INDEX `%s`, ALGORITHM=INPLACE", ddlTable(model), strings.ReplaceAll(name, "`", "``")))
	}
	return result
}

/**
* ApplyIndexes: Takes the table of the model to the FULLTEXT index of each search over text columns
* (jsql.IndexApplier): the missing ones are created and the orphan ones of jsql dropped, one ALTER TABLE at a time (InnoDB
* adds one FULLTEXT per statement) on a connection of its own, with a named lock (GET_LOCK, retried) so the services
* that start at the same time do not race. The timeout bounds the reads and the wait for the lock, not the ALTERs.
* @param model *jsql.Model, timeout ...time.Duration
* @return error
**/
func (s *Mysql) ApplyIndexes(model *jsql.Model, timeout ...time.Duration) error {
	if model.Db() == nil || model.SqlDB() == nil {
		return errors.New(jsql.MSG_DB_IS_NIL)
	}
	want := myFulltextIndexes(model)
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()
	have, err := myIndexesOf(ctx, model.SqlDB(), model)
	if err != nil {
		return err
	}
	if len(myIndexPlan(model, want, have)) == 0 {
		return nil
	}

	conn, err := model.SqlDB().Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := myLockIndexes(ctx, conn); err != nil {
		return err
	}
	defer myUnlockIndexes(conn)

	have, err = myIndexesOf(ctx, conn, model)
	if err != nil {
		return err
	}
	for _, stmt := range myIndexPlan(model, want, have) {
		if model.IsDebug {
			logs.Debug("DDL index: ", stmt)
		}
		if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}

/**
* myLockIndexes: Takes the named lock of the index DDL on conn, trying again every 200 ms until ctx ends.
* @param ctx context.Context, conn *sql.Conn
* @return error
**/
func myLockIndexes(ctx context.Context, conn *sql.Conn) error {
	for {
		var locked sql.NullInt64
		if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", myIndexLock).Scan(&locked); err != nil {
			return err
		}
		if locked.Valid && locked.Int64 == 1 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

/**
* myUnlockIndexes: Releases the named lock of the index DDL; if it cannot, the connection is discarded (closing the
* session releases the lock).
* @param conn *sql.Conn
**/
func myUnlockIndexes(conn *sql.Conn) {
	var unlocked sql.NullInt64
	err := conn.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK(?)", myIndexLock).Scan(&unlocked)
	if err != nil || !unlocked.Valid || unlocked.Int64 != 1 {
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}
