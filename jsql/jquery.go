package jsql

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
)

// keywordsCommand are the keys of a command descriptor, in the order they are checked.
var keywordsCommand = []string{
	"insert",
	"update",
	"delete",
	"upsert",
	"bulk",
}

// keywordsDDL are the keys of a definition descriptor.
var keywordsDDL = []string{
	"define",
}

// keywordsQuery are the keys that identify a query descriptor.
var keywordsQuery = []string{
	"select",
	"selects",
	"from",
}

var (
	errJsonInvalid        = errors.New("invalid sql: the descriptor has no query, command or define key")
	errJsonManyCommands   = errors.New("invalid sql: the descriptor has more than one command")
	errJsonFromRequired   = errors.New("invalid sql: from is required")
	errJsonBulkData       = errors.New("invalid sql: bulk requires data as a list of objects")
	errJsonCommandInvalid = errors.New("invalid sql: the command must be an object")
)

/**
* queryJson: Runs a query, command or definition described in JSON (see spec §8).
* @param sql et.Json
* @return et.Items, error
**/
func (s *DB) queryJson(sql et.Json) (et.Items, error) {
	return s.queryJsonTx(nil, sql)
}

/**
* queryJsonTx: Runs a query, command or definition described in JSON inside a transaction.
* The kind is chosen in a fixed order: a command key (insert, update, delete, upsert, bulk), then
* define, then a query (select or from). Keys are case-insensitive.
* @param tx *Tx, sql et.Json
* @return et.Items, error
**/
func (s *DB) queryJsonTx(tx *Tx, sql et.Json) (et.Items, error) {
	keys := map[string]string{}
	for k := range sql {
		keys[strings.ToLower(strings.TrimSpace(k))] = k
	}

	found := ""
	for _, kind := range keywordsCommand {
		if _, ok := keys[kind]; !ok {
			continue
		}
		if found != "" {
			return et.Items{}, errJsonManyCommands
		}
		found = kind
	}
	if found != "" {
		return s.parseCommand(tx, found, sql[keys[found]])
	}

	for _, kind := range keywordsDDL {
		if original, ok := keys[kind]; ok {
			return s.parseDDL(sql[original])
		}
	}

	for _, kind := range keywordsQuery {
		if _, ok := keys[kind]; ok {
			return s.parseQuery(tx, sql)
		}
	}

	return et.Items{}, errJsonInvalid
}

/**
* findModel: Resolves "schema.table" (an alias after ":" is ignored). Without schema, the table is
* looked up in every schema of the DB and must be unique.
* @param ref string
* @return *Model, error
**/
func (s *DB) findModel(ref string) (*Model, error) {
	schema, table, _, ok := splitFromRef(ref)
	if !ok {
		return nil, fmt.Errorf(MSG_INVALID_FROM, ref)
	}
	if schema != "" {
		model, err := s.GetModel(schema, table)
		if err != nil {
			return nil, fmt.Errorf(MSG_INVALID_FROM, ref)
		}
		return model, nil
	}

	var result *Model
	for name := range s.Schemas {
		model, err := s.GetModel(name, table)
		if err != nil {
			continue
		}
		if result != nil {
			return nil, fmt.Errorf("invalid from: %s exists in more than one schema, use schema.table", ref)
		}
		result = model
	}
	if result == nil {
		return nil, fmt.Errorf(MSG_INVALID_FROM, ref)
	}
	return result, nil
}

/**
* parseQuery: Runs a query descriptor: from (required) plus the keys of spec §5.4, also accepted
* with the SQL-like spelling (select, group by, order by, having, left join…) and offset.
* @param tx *Tx, sql et.Json
* @return et.Items, error
**/
func (s *DB) parseQuery(tx *Tx, sql et.Json) (et.Items, error) {
	query := normalizeQuery(sql)
	var first string
	switch v := query["from"].(type) {
	case string:
		first = v
	case []any:
		if len(v) > 0 {
			first, _ = v[0].(string)
		}
	case []string:
		if len(v) > 0 {
			first = v[0]
		}
	}
	if first == "" {
		return et.Items{}, errJsonFromRequired
	}

	// The primary model is resolved here (also without schema); loadFrom then applies the from
	// references using its schema as the default one.
	model, err := s.findModel(first)
	if err != nil {
		return et.Items{}, err
	}

	result := NewQuery(model)
	if _, err := result.loadQuery(query); err != nil {
		return et.Items{}, err
	}
	return result.AllTx(tx)
}

/**
* parseCommand: Runs a command descriptor (spec §8.1): from, data, where, limit (update, delete and
* upsert: rows it works on, 0 = all; default DB_RECORD_LIMIT up to 1000) and the JS triggers
* before_insert, after_insert, before_update, after_update, before_delete, after_delete,
* before_insert_update and after_insert_update. It returns the affected records (RETURNING).
* @param tx *Tx, kind string, value any
* @return et.Items, error
**/
func (s *DB) parseCommand(tx *Tx, kind string, value any) (et.Items, error) {
	desc, ok := toJson(value)
	if !ok {
		return et.Items{}, errJsonCommandInvalid
	}
	desc = normalizeQuery(desc)

	from := desc.Str("from")
	if from == "" {
		return et.Items{}, errJsonFromRequired
	}
	model, err := s.findModel(from)
	if err != nil {
		return et.Items{}, err
	}

	conditions, err := et.ToConditions(desc.ArrayJson("where"))
	if err != nil {
		return et.Items{}, err
	}

	var command *Command
	switch kind {
	case "insert":
		command = model.Insert(desc.Json("data"))
	case "bulk":
		data := desc.ArrayJson("data")
		if len(data) == 0 {
			return et.Items{}, errJsonBulkData
		}
		command = model.Bulk(data)
	case "update":
		command = model.Update(desc.Json("data"))
	case "delete":
		command = model.Delete()
	case "upsert":
		if len(conditions) == 0 {
			return et.Items{}, ErrUpsertWhereRequired
		}
		command = model.Upsert(desc.Json("data"))
	}
	command.Conditions = append(command.Conditions, conditions...)
	if _, ok := desc["limit"]; ok {
		command.Limit(desc.Int("limit"))
	}

	scripts := func(key string) []string {
		return desc.ArrayStr(key)
	}
	command.BeforeInserts = append(command.BeforeInserts, scripts("before_insert")...)
	command.AfterInserts = append(command.AfterInserts, scripts("after_insert")...)
	command.BeforeUpdates = append(command.BeforeUpdates, scripts("before_update")...)
	command.AfterUpdates = append(command.AfterUpdates, scripts("after_update")...)
	command.BeforeDeletes = append(command.BeforeDeletes, scripts("before_delete")...)
	command.AfterDeletes = append(command.AfterDeletes, scripts("after_delete")...)
	for _, code := range scripts("before_insert_update") {
		command.BeforeInserts = append(command.BeforeInserts, code)
		command.BeforeUpdates = append(command.BeforeUpdates, code)
	}
	for _, code := range scripts("after_insert_update") {
		command.AfterInserts = append(command.AfterInserts, code)
		command.AfterUpdates = append(command.AfterUpdates, code)
	}

	return command.ExecTx(tx)
}

/**
* parseDDL: Runs a define descriptor: the Define structure in JSON. The model is defined and
* initialized (its table is created when missing) and returned as its JSON definition.
* @param value any
* @return et.Items, error
**/
func (s *DB) parseDDL(value any) (et.Items, error) {
	bt, err := json.Marshal(value)
	if err != nil {
		return et.Items{}, err
	}
	var define Define
	if err := json.Unmarshal(bt, &define); err != nil {
		return et.Items{}, err
	}

	model, err := s.Define(define)
	if err != nil {
		return et.Items{}, err
	}
	if err := model.Init(); err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(model.ToJson())
	return result, nil
}

/**
* toJson: Converts a descriptor value (et.Json or map) to et.Json.
* @param value any
* @return et.Json, bool
**/
func toJson(value any) (et.Json, bool) {
	switch v := value.(type) {
	case et.Json:
		return v, true
	case map[string]any:
		return et.Json(v), true
	}
	return nil, false
}
