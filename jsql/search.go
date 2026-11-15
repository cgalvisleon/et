package jsql

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* TextSearch: A text search of a model (spec §6.1): a virtual field (no column, only its index) over columns and attributes
* of the SourceField, with the lang of its queries (an ISO code, es, en…, or a configuration of the engine, simple,
* spanish…) and the langs it is indexed in besides it (Langs).
**/
type TextSearch struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Lang   string   `json:"lang"`
	Langs  []string `json:"langs,omitempty"`
}

/**
* DefSearch: A search in a Define or in the json of a model: {"name": "fts", "fields": ["name", "tags"], "lang": "es",
* "langs": ["en"]}.
**/
type DefSearch struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Lang   string   `json:"lang"`
	Langs  []string `json:"langs"`
}

/**
* Indexer: A driver that creates the indexes jsql cannot leave to Load, which runs only for a new table: the indexes of
* the searches and the ones with a method (trgm). The core runs its DDL in a transaction every time a model starts
* (Init), so they also reach a table that already exists; the DDL must be idempotent. A driver that is an IndexApplier
* is not asked for it.
**/
type Indexer interface {
	LoadIndexes(model *Model, timeout ...time.Duration) (string, error)
}

/**
* IndexApplier: A driver that creates and drops the indexes of the searches and the ones with a method itself, one
* statement after the other and outside a transaction (Postgres CREATE INDEX CONCURRENTLY, MySQL ALTER TABLE …
* ALGORITHM=INPLACE): the missing and the invalid ones are created, and the search indexes jsql made that no search of
* the model uses any more are dropped. The core runs it on every Init (unless DB_SEARCH_INDEXES=off) and from
* DB.ApplySearchIndexes.
**/
type IndexApplier interface {
	ApplyIndexes(model *Model, timeout ...time.Duration) error
}

/**
* LikeSearch: A driver over the Postgres generator whose engine has neither tsvector nor pg_trgm (josefina): searches
* are rendered with LIKE and an index with method trgm is a plain one.
**/
type LikeSearch interface {
	LikeSearch() bool
}

var (
	searchName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	searchPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(->[A-Za-z_][A-Za-z0-9_]*)*$`)
	searchLang = regexp.MustCompile(`^[a-z_][a-z0-9_]*([-][a-z0-9]+)*$`)
)

// INDEX_TRGM: the method of an index that speeds up like '%x%' in Postgres (pg_trgm)
const INDEX_TRGM = "trgm"

// FULL_TEXT: the name of the search of the model (Define.FullText, the key full_text of its json and of a query)
const FULL_TEXT = "full_text"

// SEARCH_LANG_SIMPLE: the configuration of a lang jsql does not know: words as they are, without stemming
const SEARCH_LANG_SIMPLE = "simple"

// searchConfigs: the ISO 639-1 code of each text search configuration Postgres brings (pg_ts_config of Postgres 16);
// a configuration is also accepted by its own name
var searchConfigs = map[string]string{
	"ar": "arabic", "hy": "armenian", "eu": "basque", "ca": "catalan", "da": "danish", "nl": "dutch",
	"en": "english", "fi": "finnish", "fr": "french", "de": "german", "el": "greek", "hi": "hindi",
	"hu": "hungarian", "id": "indonesian", "ga": "irish", "it": "italian", "lt": "lithuanian", "ne": "nepali",
	"no": "norwegian", "nb": "norwegian", "nn": "norwegian", "pt": "portuguese", "ro": "romanian", "ru": "russian",
	"sr": "serbian", "es": "spanish", "sv": "swedish", "ta": "tamil", "tr": "turkish", "yi": "yiddish",
}

/**
* searchConfig: The text search configuration of a lang: an ISO code (es, en, pt-BR…) or a configuration by its name
* (spanish, simple…); empty is the default lang (DB_SEARCH_LANG or spanish) and an unknown one is simple. The result is
* always one of the known configurations, so it is safe in SQL.
* @param lang string
* @return string
**/
func searchConfig(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		lang = searchDefaultLang()
	}
	if lang == SEARCH_LANG_SIMPLE {
		return lang
	}
	for _, config := range searchConfigs {
		if config == lang {
			return config
		}
	}
	code, _, _ := strings.Cut(strings.ReplaceAll(lang, "_", "-"), "-")
	if config, ok := searchConfigs[code]; ok {
		return config
	}
	return SEARCH_LANG_SIMPLE
}

/**
* searchDefaultLang: The lang of a search without lang: DB_SEARCH_LANG, or spanish.
* @return string
**/
func searchDefaultLang() string {
	return strings.ToLower(strings.TrimSpace(envar.GetStr("DB_SEARCH_LANG", "spanish")))
}

/**
* searchIndexesOn: Whether Init creates and drops the search indexes (DB_SEARCH_INDEXES, on by default; off, false, 0
* or no turn it off, and DB.ApplySearchIndexes applies them by hand).
* @return bool
**/
func searchIndexesOn() bool {
	switch strings.ToLower(strings.TrimSpace(envar.GetStr("DB_SEARCH_INDEXES", "on"))) {
	case "off", "false", "0", "no":
		return false
	}
	return true
}

/**
* indexConfigs: The text search configurations a search is indexed in: the one of its lang and those of its langs,
* without repeating.
* @return []string
**/
func (s *TextSearch) indexConfigs() []string {
	result := []string{searchConfig(s.Lang)}
	for _, lang := range s.Langs {
		if config := searchConfig(lang); !slices.Contains(result, config) {
			result = append(result, config)
		}
	}
	return result
}

/**
* checkLang: The lang of a search in lowercase, or an error when it is not an ISO code or a name of a configuration
* (letters, digits, _ and -); empty stays empty.
* @param name, lang string
* @return string, error
**/
func checkLang(name, lang string) (string, error) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang != "" && !searchLang.MatchString(lang) {
		return "", fmt.Errorf(MSG_SEARCH_INVALID, name, "lang "+lang)
	}
	return lang, nil
}

/**
* defineSearch: Defines a search of the model: a virtual field name over fields (columns, or attributes of the
* SourceField), indexed by the driver (Indexer) in lang and langs. A search with the same name is replaced.
* @param name, lang string, langs []string, fields ...string
* @return *TextSearch, error
**/
func (s *Model) defineSearch(name, lang string, langs []string, fields ...string) (*TextSearch, error) {
	if !searchName.MatchString(name) {
		return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "name")
	}
	if s.indexColumn(name) != -1 {
		return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "the name is a column of the model")
	}
	lang, err := checkLang(name, lang)
	if err != nil {
		return nil, err
	}
	if lang == "" {
		lang = searchDefaultLang()
	}
	indexLangs := []string{}
	for _, item := range langs {
		item, err := checkLang(name, item)
		if err != nil {
			return nil, err
		}
		if item != "" && !slices.Contains(indexLangs, item) {
			indexLangs = append(indexLangs, item)
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "fields")
	}
	for _, field := range fields {
		if !searchPath.MatchString(field) {
			return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "field "+field)
		}
		col, ok := s.getColumn(field)
		if !ok || (col.TypeColumn != COLUMN && col.TypeColumn != ATTRIB) {
			return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "field "+field+" is not a column or an attribute")
		}
		if col.TypeColumn == COLUMN && !searchableType(col.TypeData) {
			return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "field "+field+" of type "+col.TypeData.Str())
		}
	}

	result := &TextSearch{Name: name, Fields: slices.Clone(fields), Lang: lang}
	if len(indexLangs) > 0 {
		result.Langs = indexLangs
	}
	s.muDef.Lock()
	s.Searches = withEntry(s.Searches, name, result)
	s.muDef.Unlock()
	return result, nil
}

/**
* defineSearchLangs: Sets the langs a search of the model is indexed in besides its lang (spec §6.1); lang, when it is
* not empty, also replaces the lang of its queries.
* @param name, lang string, langs ...string
* @return *TextSearch, error
**/
func (s *Model) defineSearchLangs(name, lang string, langs ...string) (*TextSearch, error) {
	search, ok := s.getSearch(name)
	if !ok {
		if name == FULL_TEXT {
			return nil, fmt.Errorf(MSG_FULLTEXT_UNDEFINED, s.Name)
		}
		return nil, fmt.Errorf(MSG_SEARCH_INVALID, name, "it is not a search of the model")
	}
	if lang == "" {
		lang = search.Lang
	}
	return s.defineSearch(name, lang, langs, search.Fields...)
}

/**
* defineFullText: Defines the search of the model, full_text, over fields; a full_text already defined keeps its lang
* and its langs.
* @param fields ...string
* @return *TextSearch, error
**/
func (s *Model) defineFullText(fields ...string) (*TextSearch, error) {
	lang, langs := "", []string(nil)
	if search, ok := s.getSearch(FULL_TEXT); ok {
		lang, langs = search.Lang, search.Langs
	}
	return s.defineSearch(FULL_TEXT, lang, langs, fields...)
}

/**
* defineFullTextOf: The full_text of a Define, after its searches: an entry of searches named full_text besides
* full_text is an error.
* @param define Define
* @return error
**/
func (s *Model) defineFullTextOf(define Define) error {
	if len(define.FullText) == 0 {
		if define.FullTextLang != "" || len(define.FullTextLangs) > 0 {
			return fmt.Errorf(MSG_FULLTEXT_UNDEFINED, s.Name)
		}
		return nil
	}
	if slices.ContainsFunc(define.Searches, func(search DefSearch) bool { return search.Name == FULL_TEXT }) {
		return fmt.Errorf(MSG_SEARCH_INVALID, FULL_TEXT, "it is defined twice, in full_text and in searches")
	}
	_, err := s.defineSearch(FULL_TEXT, define.FullTextLang, define.FullTextLangs, define.FullText...)
	return err
}

/**
* searchableType: Whether a column of type tp can be in a search: texts, numbers, booleans and JSON (an index needs an
* immutable text of it, which a date has not).
* @param tp et.TypeData
* @return bool
**/
func searchableType(tp et.TypeData) bool {
	switch tp {
	case et.DATETIME, et.BYTE, et.ARRAY_DATETIME:
		return false
	}
	return true
}

/**
* getSearch: The search of the model with that name.
* @param name string
* @return *TextSearch, bool
**/
func (s *Model) getSearch(name string) (*TextSearch, bool) {
	if s == nil {
		return nil, false
	}
	result, ok := s.Searches[name]
	return result, ok
}

/**
* searchFields: The fields a search condition looks in: the ones of a search of the model (named with the same origin
* as field: "A.fts" gives "A.name", "A.tags"…) or field itself; search is nil for a field alone.
* @param getField func(string) (*Field, bool), field string
* @return []*Field, *TextSearch, bool (false when a field does not resolve)
**/
func searchFields(getField func(string) (*Field, bool), field string) ([]*Field, *TextSearch, bool) {
	fld, ok := getField(field)
	if !ok || fld.From == nil || fld.Agg != nil {
		return nil, nil, false
	}
	if fld.TypeColumn != SEARCH {
		if fld.TypeColumn != COLUMN && fld.TypeColumn != ATTRIB {
			return nil, nil, false
		}
		return []*Field{fld}, nil, true
	}

	search, ok := fld.From.Model.getSearch(fld.Name)
	if !ok {
		return nil, nil, false
	}
	result := make([]*Field, 0, len(search.Fields))
	for _, name := range search.Fields {
		if fld.Source != "" {
			name = fld.Source + "." + name
		}
		item, ok := getField(name)
		if !ok || item.TypeColumn == SEARCH {
			return nil, nil, false
		}
		result = append(result, item)
	}
	return result, search, true
}

/**
* searchLike: The LIKE form of a search, for the engines without text search (spec §6.1): each term of the text
* (et.ToSearch) has to be in one of exprs, an excluded one in none (a NULL field does not have it) and the groups go
* joined by OR. like renders "expr LIKE pattern" in the dialect of the driver (quoting the pattern). A text without
* terms never matches (1 = 0).
* @param exprs []string, text string, like func(expr, pattern string) string
* @return string
**/
func searchLike(exprs []string, text string, like func(expr, pattern string) string) string {
	groups := []string{}
	for _, group := range et.ToSearch(text) {
		terms := []string{}
		for _, term := range group {
			pattern := et.SearchPattern(term)
			parts := make([]string, 0, len(exprs))
			for _, expr := range exprs {
				if term.Exclude {
					parts = append(parts, fmt.Sprintf("(%s IS NOT NULL AND %s)", expr, like(expr, pattern)))
				} else {
					parts = append(parts, like(expr, pattern))
				}
			}
			part := strings.Join(parts, " OR ")
			if term.Exclude {
				terms = append(terms, fmt.Sprintf("NOT (%s)", part))
			} else {
				terms = append(terms, fmt.Sprintf("(%s)", part))
			}
		}
		if len(terms) > 0 {
			groups = append(groups, "("+strings.Join(terms, " AND ")+")")
		}
	}
	if len(groups) == 0 || len(exprs) == 0 {
		return "1 = 0"
	}
	return "(" + strings.Join(groups, " OR ") + ")"
}

/**
* searchByLike: Whether the driver of the model's DB renders searches with LIKE (LikeSearch).
* @param model *Model
* @return bool
**/
func searchByLike(model *Model) bool {
	if model == nil || model.db == nil {
		return false
	}
	driver, ok := model.db.driver.(LikeSearch)
	return ok && driver.LikeSearch()
}

/**
* searchCondition: The search condition of the where of the query over the search field name (the same origin and
* search), the one whose text orders by relevance.
* @param name string
* @return *et.Condition, bool
**/
func (s *Query) searchCondition(name string) (*et.Condition, bool) {
	fld, ok := s.getField(name)
	if !ok || fld.TypeColumn != SEARCH {
		return nil, false
	}
	for _, cond := range s.Conditions {
		if cond.Operator != et.SEARCH {
			continue
		}
		other, ok := s.getField(cond.Field.String())
		if ok && other.TypeColumn == SEARCH && other.From == fld.From && other.Name == fld.Name {
			return cond, true
		}
	}
	return nil, false
}

/**
* checkSearches: A search field goes only in a search condition of the where and in the order by (relevance).
* @return error
**/
func (s *Query) checkSearches() error {
	isSearch := func(field string) bool {
		fld, ok := s.getField(field)
		return ok && fld.TypeColumn == SEARCH
	}
	for _, cond := range s.Conditions {
		if cond.Operator != et.SEARCH && isSearch(cond.Field.String()) {
			return fmt.Errorf(MSG_SEARCH_ONLY_SEARCH, cond.Field.String())
		}
	}
	for _, cond := range s.Havings {
		if isSearch(cond.Field.String()) {
			return fmt.Errorf(MSG_SEARCH_ONLY_SEARCH, cond.Field.String())
		}
	}
	for _, field := range append(slices.Clone(s.Selects), s.GroupsBy...) {
		if isSearch(field) {
			return fmt.Errorf(MSG_SEARCH_ONLY_SEARCH, field)
		}
	}
	return nil
}

/**
* loadIndexes: The indexes the driver builds on every start (Indexer, IndexApplier): searches and indexes with a
* method; nothing with DB_SEARCH_INDEXES=off.
* @param model *Model
* @return error
**/
func (s *DB) loadIndexes(model *Model) error {
	if !searchIndexesOn() {
		return nil
	}
	return s.applySearchIndexes(model)
}

/**
* applySearchIndexes: Creates the missing and invalid indexes of the searches and the ones with a method of the model,
* and drops the search indexes jsql made that no search uses any more, whatever DB_SEARCH_INDEXES says (for a
* maintenance window). A driver that is an IndexApplier runs them itself, outside a transaction; an Indexer gives the DDL
* the core runs in one.
* @param model *Model
* @return error
**/
func (s *DB) applySearchIndexes(model *Model) error {
	if s.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}
	if model == nil {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, "nil")
	}
	if applier, ok := s.driver.(IndexApplier); ok {
		return applier.ApplyIndexes(model, s.timeout)
	}
	indexer, ok := s.driver.(Indexer)
	if !ok || !model.hasLoadIndexes() {
		return nil
	}
	sql, err := indexer.LoadIndexes(model, s.timeout)
	if err != nil || sql == "" {
		return err
	}
	if model.IsDebug {
		logs.Debug("DDL indexes:\n", sql)
	}
	_, err = s.SqlTx(nil, sql)
	return err
}

/**
* hasLoadIndexes: Whether the model has searches or indexes with a method, the ones an Indexer builds.
* @return bool
**/
func (s *Model) hasLoadIndexes() bool {
	return len(s.Searches) > 0 || slices.ContainsFunc(s.Indexes, func(idx *Index) bool { return idx.Method != "" })
}

/**
* setSearch: Keeps the search of the key "search" of a query: text in the full_text of its first origin, in lang
* (empty: the lang of the full_text). It is joined to the where when the query runs (withFullText), so it does not
* depend on the order of Where, And and Or; calling it again replaces it.
* @param text string, lang ...string
* @return *Query
**/
func (s *Query) setSearch(text string, lang ...string) *Query {
	var value any = text
	if len(lang) > 0 && strings.TrimSpace(lang[0]) != "" {
		value = et.Json{"text": text, "lang": strings.TrimSpace(lang[0])}
	}
	field := FULL_TEXT
	if len(s.Froms) > 0 && s.Froms[0].As != "" {
		field = s.Froms[0].As + "." + FULL_TEXT
	}
	s.fullText = et.Search(field, value)
	return s
}

/**
* withFullText: Joins the search of the key "search" to the where while the driver renders the query: with AND to each
* group of the or (the drivers do not group, so (a OR b) AND s is a AND s OR b AND s) and, without orders, the order by
* relevance (not in a count or an exists). restore gives the query back as it was, so running it again does not repeat
* the search. Without a full_text in the model of the first origin: MSG_FULLTEXT_UNDEFINED.
* @return func(), error (restore)
**/
func (s *Query) withFullText() (func(), error) {
	if s.fullText == nil {
		return func() {}, nil
	}
	if len(s.Froms) == 0 || s.Froms[0].Model == nil {
		return nil, errors.New(MSG_QUERY_NO_FROM)
	}
	model := s.Froms[0].Model
	if _, ok := model.getSearch(FULL_TEXT); !ok {
		return nil, fmt.Errorf(MSG_FULLTEXT_UNDEFINED, model.Name)
	}

	conditions, orders := s.Conditions, s.OrdersBy
	s.Conditions = rollupConditions([]*et.Condition{s.fullText}, conditions)
	if len(orders) == 0 && !s.IsCount && !s.IsExists {
		s.OrdersBy = []*Index{{Name: s.fullText.Field.String(), Sorted: false}}
	}
	return func() {
		s.Conditions, s.OrdersBy = conditions, orders
	}, nil
}
