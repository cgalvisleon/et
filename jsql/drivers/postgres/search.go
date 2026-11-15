package postgres

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
* pgSearchText: The text of a field a search looks in: a text column as it is, another column cast to text, an
* attribute read as text (->>). alias qualifies it; the index is built without it (the planner compares the expression,
* not the alias).
* @param fld *jsql.Field, alias string
* @return string
**/
func pgSearchText(fld *jsql.Field, alias string) string {
	if fld.TypeColumn == jsql.ATTRIB {
		return pgAttribPath(fld.From.Model, fld.Name, alias)
	}
	ref := fld.Name
	if alias != "" {
		ref = alias + "." + fld.Name
	}
	switch fld.TypeData {
	case et.TEXT, et.KEY, et.MEMO:
		return ref
	}
	return ref + "::text"
}

/**
* pgSearchVector: The tsvector of a search: to_tsvector('config', COALESCE(a, '') || ' ' || COALESCE(b, '')), with the
* configuration of lang (jsql.SearchConfig: always a known one). The index of the search in that configuration
* (pgSearchIndexes) is built over this same expression, so the query must render it the same way.
* @param fields []*jsql.Field, lang, alias string
* @return string
**/
func pgSearchVector(fields []*jsql.Field, lang, alias string) string {
	parts := make([]string, 0, len(fields))
	for _, fld := range fields {
		parts = append(parts, fmt.Sprintf("COALESCE(%s, '')", pgSearchText(fld, alias)))
	}
	return fmt.Sprintf("to_tsvector(%s::regconfig, %s)", pgQuoteString(jsql.SearchConfig(lang)), strings.Join(parts, " || ' ' || "))
}

/**
* pgSearchQuery: The tsquery of a search text in the web syntax (websearch_to_tsquery), always quoted, with the
* configuration of lang.
* @param lang string, text any
* @return string
**/
func pgSearchQuery(lang string, text any) string {
	return fmt.Sprintf("websearch_to_tsquery(%s::regconfig, %s)", pgQuoteString(jsql.SearchConfig(lang)), pgQuoteString(text))
}

/**
* pgSearchOf: The fields and the lang of a search condition: the lang of the condition ({"text", "lang"}) or else the
* one of the search of the model, or the default lang (jsql.SearchLang) for a field alone.
* @param getField func(string) (*jsql.Field, bool), field, lang string
* @return []*jsql.Field, string, bool
**/
func pgSearchOf(getField func(string) (*jsql.Field, bool), field, lang string) ([]*jsql.Field, string, bool) {
	fields, search, ok := jsql.SearchFields(getField, field)
	if !ok || len(fields) == 0 {
		return nil, "", false
	}
	if lang == "" && search != nil {
		lang = search.Lang
	}
	return fields, lang, true
}

/**
* pgSearchExpr: Renders a search condition: the tsvector of its fields @@ the tsquery of the text; with a driver that
* has no text search (jsql.SearchByLike, josefina), the LIKE form. A field that does not resolve never matches.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition
* @return string
**/
func pgSearchExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition) string {
	text, lang, isText := jsql.SearchOf(cond)
	fields, lang, ok := pgSearchOf(getField, cond.Field.String(), lang)
	if !ok || !isText {
		return "1 = 0"
	}
	if jsql.SearchByLike(fields[0].From.Model) {
		exprs := make([]string, 0, len(fields))
		for _, fld := range fields {
			exprs = append(exprs, pgSearchText(fld, fld.From.As))
		}
		return jsql.SearchLike(exprs, text, func(expr, pattern string) string {
			return fmt.Sprintf("%s ILIKE %s", expr, pgQuoteString(pattern))
		})
	}
	return fmt.Sprintf("%s @@ %s", pgSearchVector(fields, lang, fields[0].From.As), pgSearchQuery(lang, text))
}

/**
* pgSearchRank: The ORDER BY of a search field: ts_rank of its vector against the text of the search condition of the
* where, in its lang; false without that condition or with a driver that searches with LIKE.
* @param query *jsql.Query, name string
* @return string, bool
**/
func pgSearchRank(query *jsql.Query, name string) (string, bool) {
	cond, ok := query.SearchCondition(name)
	if !ok {
		return "", false
	}
	text, lang, isText := jsql.SearchOf(cond)
	fields, lang, ok := pgSearchOf(query.GetField, cond.Field.String(), lang)
	if !ok || !isText || jsql.SearchByLike(fields[0].From.Model) {
		return "", false
	}
	return fmt.Sprintf("ts_rank(%s, %s)", pgSearchVector(fields, lang, fields[0].From.As), pgSearchQuery(lang, text)), true
}

// pgMaxIdent: the longest identifier Postgres keeps (NAMEDATALEN - 1); a longer one is cut
const pgMaxIdent = 63

// pgIndexLock: the session lock that serializes the index DDL of the services that start at the same time; another
// key than ddlLock, so a Load does not wait for an index build
const pgIndexLock = "et.jsql.indexes"

// pgSearchIndexName: the name of a search index jsql made: <table>_<search>_fts_<hash of its expression>
var pgSearchIndexName = regexp.MustCompile(`_fts_[0-9a-f]{8}$`)

/**
* pgIndex: An index the model wants: its name (lowercase, as Postgres keeps an unquoted one) and its CREATE.
**/
type pgIndex struct {
	name   string
	create string
	trgm   bool
}

/**
* pgHaveIndex: An index of the table in the catalog: valid is false for one a concurrent build left half done, search
* whether it has the shape of a search index of jsql (name with the hash, GIN over to_tsvector).
**/
type pgHaveIndex struct {
	valid  bool
	search bool
}

/**
* pgIndexName: The name of an index of the table with a suffix, cut to the 63 bytes Postgres keeps (the cut is in the
* prefix, so the suffix, with the hash, stays) and in lowercase.
* @param prefix, suffix string
* @return string
**/
func pgIndexName(prefix, suffix string) string {
	if len(prefix)+len(suffix) > pgMaxIdent {
		prefix = prefix[:pgMaxIdent-len(suffix)]
	}
	return strings.ToLower(prefix + suffix)
}

/**
* pgSearchIndexes: The GIN index of each search of the model in each of its langs (jsql.SearchIndexConfigs), over the
* same tsvector the queries use; its name has a hash of the expression, so another field or lang is another index.
* @param model *jsql.Model
* @return []pgIndex
**/
func pgSearchIndexes(model *jsql.Model) []pgIndex {
	names := make([]string, 0, len(model.Searches))
	for name := range model.Searches {
		names = append(names, name)
	}
	slices.Sort(names)

	base := strings.ReplaceAll(model.Table, ".", "_")
	result := []pgIndex{}
	for _, name := range names {
		fields, _, ok := pgSearchOf(model.GetField, name, "")
		if !ok {
			continue
		}
		for _, config := range jsql.SearchIndexConfigs(model.Searches[name]) {
			expr := pgSearchVector(fields, config, "")
			h := fnv.New32a()
			h.Write([]byte(expr))
			index := pgIndexName(base+"_"+name, fmt.Sprintf("_fts_%08x", h.Sum32()))
			result = append(result, pgIndex{
				name:   index,
				create: fmt.Sprintf("CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING GIN (%s);", index, model.Table, expr),
			})
		}
	}
	return result
}

/**
* pgTrgmIndexes: The pg_trgm indexes (method trgm), so like '%x%' over the field can use an index. A field that is not
* a column or an attribute is skipped.
* @param model *jsql.Model
* @return []pgIndex
**/
func pgTrgmIndexes(model *jsql.Model) []pgIndex {
	base := strings.ReplaceAll(model.Table, ".", "_")
	result := []pgIndex{}
	for _, idx := range model.Indexes {
		if idx.Method != jsql.INDEX_TRGM {
			continue
		}
		fld, ok := model.GetField(idx.Name)
		if !ok || (fld.TypeColumn != jsql.COLUMN && fld.TypeColumn != jsql.ATTRIB) {
			continue
		}
		index := pgIndexName(base+"_"+strings.ReplaceAll(idx.Name, "->", "_"), "_trgm")
		result = append(result, pgIndex{
			name:   index,
			create: fmt.Sprintf("CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING GIN ((%s) gin_trgm_ops);", index, model.Table, pgSearchText(fld, "")),
			trgm:   true,
		})
	}
	return result
}

/**
* pgQueryer: A *sql.DB or the *sql.Conn of the lock.
**/
type pgQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

/**
* pgIndexesOf: The indexes of the table of the model in the catalog, by name.
* @param ctx context.Context, db pgQueryer, model *jsql.Model
* @return map[string]pgHaveIndex, error
**/
func pgIndexesOf(ctx context.Context, db pgQueryer, model *jsql.Model) (map[string]pgHaveIndex, error) {
	schema, table, ok := strings.Cut(model.Table, ".")
	if !ok {
		schema, table = "public", model.Table
	}
	rows, err := db.QueryContext(ctx, `SELECT c.relname, i.indisvalid, am.amname = 'gin' AND pg_get_indexdef(i.indexrelid) LIKE '%to_tsvector(%'
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
JOIN pg_class t ON t.oid = i.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_am am ON am.oid = c.relam
WHERE n.nspname = $1 AND t.relname = $2`, strings.ToLower(schema), strings.ToLower(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]pgHaveIndex{}
	for rows.Next() {
		var name string
		var valid, tsvector bool
		if err := rows.Scan(&name, &valid, &tsvector); err != nil {
			return nil, err
		}
		result[name] = pgHaveIndex{valid: valid, search: tsvector && pgSearchIndexName.MatchString(name)}
	}
	return result, rows.Err()
}

/**
* pgIndexPlan: The statements that take the table from the indexes it has to the ones the model wants, one by one and
* outside a transaction: an index that is missing is created, one left invalid by an interrupted concurrent build is
* dropped and created again, and a search index of jsql (pgHaveIndex.search) that the model does not want any more is
* dropped. Any other index of the table is never touched.
* @param schema string, want []pgIndex, have map[string]pgHaveIndex
* @return []string
**/
func pgIndexPlan(schema string, want []pgIndex, have map[string]pgHaveIndex) []string {
	drop := func(name string) string {
		return fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s.%s;", pgIdent(schema), pgIdent(name))
	}
	result, wanted, extension := []string{}, map[string]bool{}, false
	for _, idx := range want {
		wanted[idx.name] = true
		current, ok := have[idx.name]
		if ok && current.valid {
			continue
		}
		if idx.trgm && !extension {
			result = append(result, "CREATE EXTENSION IF NOT EXISTS pg_trgm;")
			extension = true
		}
		if ok {
			result = append(result, drop(idx.name))
		}
		result = append(result, idx.create)
	}

	orphans := []string{}
	for name, current := range have {
		if current.search && !wanted[name] {
			orphans = append(orphans, name)
		}
	}
	slices.Sort(orphans)
	for _, name := range orphans {
		result = append(result, drop(name))
	}
	return result
}

/**
* ApplyIndexes: Takes the table of the model to the GIN index of each search in each lang and the pg_trgm ones
* (jsql.IndexApplier), without blocking its writes: CREATE INDEX CONCURRENTLY and DROP INDEX CONCURRENTLY, which cannot go
* in a transaction, one after the other on a connection of their own. When there is something to do it takes a session
* lock (pg_try_advisory_lock, retried: a session waiting in pg_advisory_lock would hold a snapshot that the concurrent
* build of the other one waits for) and reads the catalog again. The timeout bounds the reads and the wait for the
* lock, not the builds, so a big table is never left with an invalid index by the timeout.
* @param model *jsql.Model, timeout ...time.Duration
* @return error
**/
func (s *Postgres) ApplyIndexes(model *jsql.Model, timeout ...time.Duration) error {
	if model.Db() == nil || model.SqlDB() == nil {
		return errors.New(jsql.MSG_DB_IS_NIL)
	}
	want := append(pgSearchIndexes(model), pgTrgmIndexes(model)...)
	schema, _, ok := strings.Cut(model.Table, ".")
	if !ok {
		schema = "public"
	}
	schema = strings.ToLower(schema)

	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()
	have, err := pgIndexesOf(ctx, model.SqlDB(), model)
	if err != nil {
		return err
	}
	if len(pgIndexPlan(schema, want, have)) == 0 {
		return nil
	}

	conn, err := model.SqlDB().Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := pgLockIndexes(ctx, conn); err != nil {
		return err
	}
	defer pgUnlockIndexes(conn)

	have, err = pgIndexesOf(ctx, conn, model)
	if err != nil {
		return err
	}
	for _, stmt := range pgIndexPlan(schema, want, have) {
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
* pgLockIndexes: Takes the session lock of the index DDL on conn, trying again every 200 ms until ctx ends.
* @param ctx context.Context, conn *sql.Conn
* @return error
**/
func pgLockIndexes(ctx context.Context, conn *sql.Conn) error {
	for {
		var locked bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", pgIndexLock).Scan(&locked); err != nil {
			return err
		}
		if locked {
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
* pgUnlockIndexes: Releases the session lock of the index DDL; if it cannot, the connection is discarded (closing the
* session releases the lock) instead of going back to the pool with it.
* @param conn *sql.Conn
**/
func pgUnlockIndexes(conn *sql.Conn) {
	var unlocked bool
	err := conn.QueryRowContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", pgIndexLock).Scan(&unlocked)
	if err != nil || !unlocked {
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}
