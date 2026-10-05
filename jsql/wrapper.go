package jsql

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* wrapRef: Key under which every wrapped object keeps its Go value, so it can be passed back to Go
* (a model to join, a tx to run in).
**/
const wrapRef = "__ref"

/**
* wrapper: Returns the bindings a goja (jrex) instance gets to use jsql from JavaScript: db (the model's
* database), newTx and jsql (constructors, conditions and helpers). Every *DB, *Model, *Query, *Command,
* *Tx and *Series is exposed as an object with camelCase methods; builders return the wrapped object so
* calls chain, et.Items/et.Item come back as JSON, and a Go error is thrown as a JavaScript exception.
* @param db *DB
* @return map[string]any
**/
func wrapper(db *DB) map[string]any {
	return map[string]any{
		"db":    wrapDB(db),
		"newTx": func() map[string]any { return wrapTx(NewTx()) },
		"jsql":  wrapPackage(),
	}
}

/**
* unwrap: Returns the Go value of a wrapped object, or value itself when it is not wrapped.
* @param value any
* @return any
**/
func unwrap(value any) any {
	if m, ok := value.(map[string]any); ok {
		return m[wrapRef]
	}
	return value
}

/**
* toTx: Returns the *Tx of a wrapped tx, a *Tx, or nil (no transaction).
* @param value any
* @return *Tx
**/
func toTx(value any) *Tx {
	tx, _ := unwrap(value).(*Tx)
	return tx
}

/**
* toModel: Returns the *Model of a wrapped model or a *Model, or nil.
* @param value any
* @return *Model
**/
func toModel(value any) *Model {
	model, _ := unwrap(value).(*Model)
	return model
}

/**
* toDB: Returns the *DB of a wrapped db or a *DB, or nil.
* @param value any
* @return *DB
**/
func toDB(value any) *DB {
	db, _ := unwrap(value).(*DB)
	return db
}

/**
* toDefine: Converts a define descriptor (its JSON) into a Define.
* @param value et.Json
* @return Define, error
**/
func toDefine(value et.Json) (Define, error) {
	var result Define
	bt, err := json.Marshal(value)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(bt, &result)
	return result, err
}

/**
* toJs: Converts et.Json values (also nested, and inside slices) into plain maps, since goja exposes
* the methods of et.Json instead of its keys.
* @param value any
* @return any
**/
func toJs(value any) any {
	switch v := value.(type) {
	case et.Json:
		return toJsMap(v)
	case map[string]any:
		return toJsMap(v)
	case []et.Json:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = toJsMap(item)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = toJs(item)
		}
		return result
	}
	return value
}

/**
* toJsMap: Returns a copy of value as a plain map, converting its values with toJs.
* @param value map[string]any
* @return map[string]any
**/
func toJsMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	result := make(map[string]any, len(value))
	for k, v := range value {
		result[k] = toJs(v)
	}
	return result
}

/**
* jsonFn: Wraps a func returning et.Json so JavaScript gets a plain object.
* @param fn func() et.Json
* @return func() map[string]any
**/
func jsonFn(fn func() et.Json) func() map[string]any {
	return func() map[string]any { return toJsMap(fn()) }
}

/**
* itemsJson: Returns the et.Items of a call as JSON.
* @param items et.Items, err error
* @return et.Json, error
**/
func itemsJson(items et.Items, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	return toJsMap(items.ToJson()), nil
}

/**
* itemJson: Returns the et.Item of a call as JSON.
* @param item et.Item, err error
* @return et.Json, error
**/
func itemJson(item et.Item, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	return toJsMap(item.ToJson()), nil
}

/**
* indexJson: Returns an index as JSON, or nil.
* @param index *Index
* @return any
**/
func indexJson(index *Index) any {
	if index == nil {
		return nil
	}
	return map[string]any{"name": index.Name, "sorted": index.Sorted}
}

/**
* columnJson: Returns a column as JSON, or nil.
* @param column *Column
* @return any
**/
func columnJson(column *Column) any {
	if column == nil {
		return nil
	}
	return toJsMap(column.ToJson())
}

/**
* wrapPackage: Returns the package-level functions of jsql.
* @return map[string]any
**/
func wrapPackage() map[string]any {
	return map[string]any{
		"load": func() (map[string]any, error) {
			db, err := Load()
			if err != nil {
				return nil, err
			}
			return wrapDB(db), nil
		},
		"loadTo": func(dbName string, hostName ...string) (map[string]any, error) {
			db, err := LoadTo(dbName, hostName...)
			if err != nil {
				return nil, err
			}
			return wrapDB(db), nil
		},
		"loadDb": func(params et.Json) (map[string]any, error) {
			db, err := LoadDb(params)
			if err != nil {
				return nil, err
			}
			return wrapDB(db), nil
		},
		// timeoutMs: milliseconds; 0 or none never fails by timeout
		"dropDB": func(db any, timeoutMs ...int64) error {
			timeout := []time.Duration{}
			if len(timeoutMs) > 0 {
				timeout = append(timeout, time.Duration(timeoutMs[0])*time.Millisecond)
			}
			return DropDB(toDB(db), timeout...)
		},
		"newTx": func() map[string]any { return wrapTx(NewTx()) },
		"newQuery": func(model any, as ...string) map[string]any {
			return wrapQuery(NewQuery(toModel(model), as...))
		},
		"defineSeries": func(db any, schema string) (map[string]any, error) {
			series, err := DefineSeries(toDB(db), schema)
			if err != nil {
				return nil, err
			}
			return wrapSeries(series), nil
		},
		"detailKeys": DetailKeys,
		"rollupKeys": RollupKeys,
		"where":      Where,
		"and":        And,
		"or":         Or,
		"eq":         Eq,
		"neg":        Neg,
		"less":       Less,
		"lessEq":     LessEq,
		"more":       More,
		"moreEq":     MoreEq,
		"like":       Like,
		"in":         In,
		"notIn":      NotIn,
		"is":         Is,
		"isNot":      IsNot,
		"null":       Null,
		"notNull":    NotNull,
		"between":    Between,
		"notBetween": NotBetween,
		"addAuditLog": func(auditLog []et.Json, userId, action string) any {
			return toJs(AddAuditLog(auditLog, userId, action))
		},
		"statusList":      StatusList,
		"sqlParse":        SQLParse,
		"quoted":          Quoted,
		"escapeSQLString": EscapeSQLString,
		"jsonString":      JsonString,
		"argWhitAs":       ArgWhitAs,
		"argWhitSchema":   ArgWhitSchema,
	}
}

/**
* wrapDB: Exposes a *DB to JavaScript.
* @param db *DB
* @return map[string]any
**/
func wrapDB(db *DB) map[string]any {
	if db == nil {
		return nil
	}

	model := func(model *Model, err error) (map[string]any, error) {
		if err != nil {
			return nil, err
		}
		return wrapModel(model), nil
	}

	return map[string]any{
		wrapRef:    db,
		"toJson":   jsonFn(db.ToJson),
		"init":     db.Init,
		"close":    db.Close,
		"setDebug": db.SetDebug,
		"debug":    db.Debug,
		"sql": func(query string, args ...any) (map[string]any, error) {
			return itemsJson(db.Sql(query, args...))
		},
		"sqlTx": func(tx any, query string, args ...any) (map[string]any, error) {
			return itemsJson(db.SqlTx(toTx(tx), query, args...))
		},
		"query": func(query et.Json) (map[string]any, error) {
			return itemsJson(db.Query(query))
		},
		"queryTx": func(tx any, query et.Json) (map[string]any, error) {
			return itemsJson(db.QueryTx(toTx(tx), query))
		},
		"newModel": func(schema, name string, version int, userId string) map[string]any {
			return wrapModel(db.NewModel(schema, name, version, userId))
		},
		"removeModel": db.RemoveModel,
		"getModel": func(schema, name string) (map[string]any, error) {
			return model(db.GetModel(schema, name))
		},
		"define": func(define et.Json) (map[string]any, error) {
			def, err := toDefine(define)
			if err != nil {
				return nil, err
			}
			return model(db.Define(def))
		},
		"defineModel": func(schema, name string, version int, userId string) (map[string]any, error) {
			return model(db.DefineModel(schema, name, version, userId))
		},
		"defineTenantModel": func(schema, name string, version int, userId string) (map[string]any, error) {
			return model(db.DefineTenantModel(schema, name, version, userId))
		},
		"defineProjectModel": func(schema, name string, version int, userId string) (map[string]any, error) {
			return model(db.DefineProjectModel(schema, name, version, userId))
		},
		"onAuditLog": db.OnAuditLog,
	}
}

/**
* wrapModel: Exposes a *Model to JavaScript.
* @param m *Model
* @return map[string]any
**/
func wrapModel(m *Model) map[string]any {
	if m == nil {
		return nil
	}

	var self map[string]any
	chain := func(*Model) map[string]any { return self }
	related := func(model *Model, err error) (map[string]any, error) {
		if err != nil {
			return nil, err
		}
		return wrapModel(model), nil
	}

	self = map[string]any{
		wrapRef:  m,
		"toJson": jsonFn(m.ToJson),
		"debug":  func() map[string]any { return chain(m.Debug()) },
		"init":   m.Init,
		"stricted": func() map[string]any {
			m.Stricted()
			return self
		},
		"db": func() map[string]any { return wrapDB(m.Db()) },
		"setDb": func(db any) map[string]any {
			m.SetDb(toDB(db))
			return self
		},
		"getModel": func(schema, name string) (map[string]any, error) {
			return related(m.GetModel(schema, name))
		},
		"addAuditLog": m.AddAuditLog,
		// Definition
		"defineSource":   func() any { return columnJson(m.DefineSource()) },
		"defineIdxField": func() any { return indexJson(m.DefineIdxField()) },
		"defineIndex": func(name string, tp et.TypeData, deFault any) any {
			return indexJson(m.DefineIndex(name, tp, deFault))
		},
		"definePrimaryKey": func(name string, tp et.TypeData, deFault any) any {
			return indexJson(m.DefinePrimaryKey(name, tp, deFault))
		},
		"defineUnique": func(name string, tp et.TypeData, deFault any) any {
			return indexJson(m.DefineUnique(name, tp, deFault))
		},
		"defineRequired": func(name string, tp et.TypeData, deFault any) any {
			return indexJson(m.DefineRequired(name, tp, deFault))
		},
		"defineColumn": func(name string, tp et.TypeData, deFault any) any {
			return columnJson(m.DefineColumn(name, tp, deFault))
		},
		"defineAttrib": func(name string, tp et.TypeData, deFault any) any {
			return columnJson(m.DefineAttrib(name, tp, deFault))
		},
		"defineForeignKeys": func(to any, keys map[string]string, onDeleteCascade, onUpdateCascade bool) any {
			detail := m.DefineForeignKeys(toModel(to), keys, onDeleteCascade, onUpdateCascade)
			if detail == nil {
				return nil
			}
			return toJsMap(detail.Ref())
		},
		"defineOmitUpdate": func(names ...string) map[string]any { return chain(m.DefineOmitUpdate(names...)) },
		"defineHidden": func(names ...string) map[string]any {
			m.DefineHidden(names...)
			return self
		},
		"defineModel": func() map[string]any { return chain(m.DefineModel()) },
		"defineRollup": func(name string, to any, keys map[string]string, selects []string, operation RollupOperation) (*Rollups, error) {
			return m.DefineRollup(name, toModel(to), keys, selects, operation)
		},
		"defineDetail": func(name string, keys map[string]string, rows int, selects ...string) (map[string]any, error) {
			return related(m.DefineDetail(name, keys, rows, selects...))
		},
		"defineMaster": func(name string, to any, keys, toKeys map[string]string, selects []string, rows ...int) (map[string]any, error) {
			return related(m.DefineMaster(name, toModel(to), keys, toKeys, selects, rows...))
		},
		"defineCalcFunc": func(name string, calc CalcFunction) map[string]any {
			return chain(m.DefineCalcFunc(name, calc))
		},
		"defineCalc": func(name, script string) map[string]any { return chain(m.DefineCalc(name, script)) },
		"defineBeforeInsert": func(name, code string, version int) map[string]any {
			return chain(m.DefineBeforeInsert(name, code, version))
		},
		"defineBeforeUpdate": func(name, code string, version int) map[string]any {
			return chain(m.DefineBeforeUpdate(name, code, version))
		},
		"defineBeforeDelete": func(name, code string, version int) map[string]any {
			return chain(m.DefineBeforeDelete(name, code, version))
		},
		"defineAfterInsert": func(name, code string, version int) map[string]any {
			return chain(m.DefineAfterInsert(name, code, version))
		},
		"defineAfterUpdate": func(name, code string, version int) map[string]any {
			return chain(m.DefineAfterUpdate(name, code, version))
		},
		"defineAfterDelete": func(name, code string, version int) map[string]any {
			return chain(m.DefineAfterDelete(name, code, version))
		},
		// Triggers (JavaScript functions: (tx, old, new) => {})
		"beforeInsert":         func(fn TriggerFunction) map[string]any { return chain(m.BeforeInsert(fn)) },
		"beforeUpdate":         func(fn TriggerFunction) map[string]any { return chain(m.BeforeUpdate(fn)) },
		"beforeDelete":         func(fn TriggerFunction) map[string]any { return chain(m.BeforeDelete(fn)) },
		"beforeInsertOrUpdate": func(fn TriggerFunction) map[string]any { return chain(m.BeforeInsertOrUpdate(fn)) },
		"afterInsert":          func(fn TriggerFunction) map[string]any { return chain(m.AfterInsert(fn)) },
		"afterUpdate":          func(fn TriggerFunction) map[string]any { return chain(m.AfterUpdate(fn)) },
		"afterDelete":          func(fn TriggerFunction) map[string]any { return chain(m.AfterDelete(fn)) },
		"afterInsertOrUpdate":  func(fn TriggerFunction) map[string]any { return chain(m.AfterInsertOrUpdate(fn)) },
		// Lookups (null when missing)
		"getCalcFunc": func(name string) any {
			if fn, ok := m.GetCalcFunc(name); ok {
				return fn
			}
			return nil
		},
		"detail": func(name string) map[string]any {
			if model, ok := m.Detail(name); ok {
				return wrapModel(model)
			}
			return nil
		},
		"master": func(name string) map[string]any {
			if query, ok := m.Master(name); ok {
				return wrapQuery(query)
			}
			return nil
		},
		"bridge": func(name string) map[string]any {
			if model, ok := m.Bridge(name); ok {
				return wrapModel(model)
			}
			return nil
		},
		"getColumn": func(name string) any {
			if column, ok := m.GetColumn(name); ok {
				return columnJson(column)
			}
			return nil
		},
		"getField": func(name string) any {
			if field, ok := m.GetField(name); ok {
				return field
			}
			return nil
		},
		"getFrom": func() any {
			if from := m.GetFrom(); from != nil {
				return toJsMap(from.Ref())
			}
			return nil
		},
		// Queries
		"as": func(as ...string) map[string]any { return wrapQuery(m.As(as...)) },
		"join": func(to any, as string, on []*et.Condition) map[string]any {
			return wrapQuery(m.Join(toModel(to), as, on))
		},
		"select": func(fields ...string) map[string]any { return wrapQuery(m.Select(fields...)) },
		"calc":   func(fields ...string) map[string]any { return wrapQuery(m.Calc(fields...)) },
		"where":  func(cond *et.Condition) map[string]any { return wrapQuery(m.Where(cond)) },
		"count":  m.Count,
		"first":  func(n int) (map[string]any, error) { return itemsJson(m.First(n)) },
		"queryTx": func(tx any, query et.Json) map[string]any {
			return wrapQuery(m.QueryTx(toTx(tx), query))
		},
		"query": func(query et.Json) map[string]any { return wrapQuery(m.Query(query)) },
		// Commands
		"insert": func(data et.Json) map[string]any { return wrapCommand(m.Insert(data)) },
		"bulk":   func(data []et.Json) map[string]any { return wrapCommand(m.Bulk(data)) },
		"update": func(data et.Json) map[string]any { return wrapCommand(m.Update(data)) },
		"delete": func() map[string]any { return wrapCommand(m.Delete()) },
		"upsert": func(data et.Json) map[string]any { return wrapCommand(m.Upsert(data)) },
	}
	return self
}

/**
* wrapQuery: Exposes a *Query to JavaScript.
* @param q *Query
* @return map[string]any
**/
func wrapQuery(q *Query) map[string]any {
	if q == nil {
		return nil
	}

	var self map[string]any
	chain := func(*Query) map[string]any { return self }
	self = map[string]any{
		wrapRef:  q,
		"toJson": jsonFn(q.ToJson),
		"debug":  func() map[string]any { return chain(q.Debug()) },
		"test":   func() map[string]any { return chain(q.Test()) },
		"getField": func(name string) any {
			if field, ok := q.GetField(name); ok {
				return field
			}
			return nil
		},
		"getSelectField": func(name string) any {
			if field, ok := q.GetSelectField(name); ok {
				return field
			}
			return nil
		},
		"join": func(model any, as string, on []*et.Condition) map[string]any {
			return chain(q.Join(toModel(model), as, on))
		},
		"leftJoin": func(model any, as string, on []*et.Condition) map[string]any {
			return chain(q.LeftJoin(toModel(model), as, on))
		},
		"rightJoin": func(model any, as string, on []*et.Condition) map[string]any {
			return chain(q.RightJoin(toModel(model), as, on))
		},
		"fullJoin": func(model any, as string, on []*et.Condition) map[string]any {
			return chain(q.FullJoin(toModel(model), as, on))
		},
		"select":       func(fields ...string) map[string]any { return chain(q.Select(fields...)) },
		"calc":         func(fields ...string) map[string]any { return chain(q.Calc(fields...)) },
		"detail":       func(fields ...string) map[string]any { return chain(q.Detail(fields...)) },
		"master":       func(fields ...string) map[string]any { return chain(q.Master(fields...)) },
		"hidden":       func(fields ...string) map[string]any { return chain(q.Hidden(fields...)) },
		"addCondition": func(cond *et.Condition) map[string]any { return chain(q.AddCondition(cond)) },
		"where":        func(cond *et.Condition) map[string]any { return chain(q.Where(cond)) },
		"and":          func(cond *et.Condition) map[string]any { return chain(q.And(cond)) },
		"or":           func(cond *et.Condition) map[string]any { return chain(q.Or(cond)) },
		"groupBy":      func(fields ...string) map[string]any { return chain(q.GroupBy(fields...)) },
		"having":       func(cond *et.Condition) map[string]any { return chain(q.Having(cond)) },
		"page":         func(page int) map[string]any { return chain(q.Page(page)) },
		"orderBy": func(field string, sorted ...bool) map[string]any {
			return chain(q.OrderBy(field, sorted...))
		},
		"allTx":   func(tx any) (map[string]any, error) { return itemsJson(q.AllTx(toTx(tx))) },
		"all":     func() (map[string]any, error) { return itemsJson(q.All()) },
		"oneTx":   func(tx any) (map[string]any, error) { return itemJson(q.OneTx(toTx(tx))) },
		"one":     func() (map[string]any, error) { return itemJson(q.One()) },
		"firstTx": func(tx any, n int) (map[string]any, error) { return itemsJson(q.FirstTx(toTx(tx), n)) },
		"first":   func(n int) (map[string]any, error) { return itemsJson(q.First(n)) },
		"limitTx": func(tx any, page, rows int) (map[string]any, error) {
			return itemsJson(q.LimitTx(toTx(tx), page, rows))
		},
		"limit":    func(page, rows int) (map[string]any, error) { return itemsJson(q.Limit(page, rows)) },
		"existsTx": func(tx any) (bool, error) { return q.ExistsTx(toTx(tx)) },
		"exists":   q.Exists,
		"countTx":  func(tx any) (int, error) { return q.CountTx(toTx(tx)) },
		"count":    q.Count,
	}
	return self
}

/**
* wrapCommand: Exposes a *Command to JavaScript.
* @param c *Command
* @return map[string]any
**/
func wrapCommand(c *Command) map[string]any {
	if c == nil {
		return nil
	}

	var self map[string]any
	chain := func(*Command) map[string]any { return self }
	self = map[string]any{
		wrapRef:  c,
		"toJson": jsonFn(c.ToJson),
		"debug":  func() map[string]any { return chain(c.Debug()) },
		"test":   func() map[string]any { return chain(c.Test()) },
		"where":  func(cond *et.Condition) map[string]any { return chain(c.Where(cond)) },
		"and":    func(cond *et.Condition) map[string]any { return chain(c.And(cond)) },
		"or":     func(cond *et.Condition) map[string]any { return chain(c.Or(cond)) },
		"return": func(fields ...string) map[string]any { return chain(c.Return(fields...)) },
		"limit":  func(rows int) map[string]any { return chain(c.Limit(rows)) },
		"execTx": func(tx any) (map[string]any, error) { return itemsJson(c.ExecTx(toTx(tx))) },
		"exec":   func() (map[string]any, error) { return itemsJson(c.Exec()) },
		"oneTx":  func(tx any) (map[string]any, error) { return itemJson(c.OneTx(toTx(tx))) },
		"one":    func() (map[string]any, error) { return itemJson(c.One()) },
		// Triggers (JavaScript functions: (tx, old, new) => {})
		"beforeInsert":         func(fn TriggerFunction) map[string]any { return chain(c.BeforeInsert(fn)) },
		"beforeUpdate":         func(fn TriggerFunction) map[string]any { return chain(c.BeforeUpdate(fn)) },
		"beforeDelete":         func(fn TriggerFunction) map[string]any { return chain(c.BeforeDelete(fn)) },
		"beforeInsertOrUpdate": func(fn TriggerFunction) map[string]any { return chain(c.BeforeInsertOrUpdate(fn)) },
		"afterInsert":          func(fn TriggerFunction) map[string]any { return chain(c.AfterInsert(fn)) },
		"afterUpdate":          func(fn TriggerFunction) map[string]any { return chain(c.AfterUpdate(fn)) },
		"afterInsertOrUpdate":  func(fn TriggerFunction) map[string]any { return chain(c.AfterInsertOrUpdate(fn)) },
		"afterDelete":          func(fn TriggerFunction) map[string]any { return chain(c.AfterDelete(fn)) },
	}
	return self
}

/**
* wrapTx: Exposes a *Tx to JavaScript.
* @param tx *Tx
* @return map[string]any
**/
func wrapTx(tx *Tx) map[string]any {
	return map[string]any{
		wrapRef:    tx,
		"commit":   tx.Commit,
		"rollback": tx.Rollback,
	}
}

/**
* wrapSeries: Exposes a *Series to JavaScript.
* @param s *Series
* @return map[string]any
**/
func wrapSeries(s *Series) map[string]any {
	if s == nil {
		return nil
	}

	return map[string]any{
		wrapRef:        s,
		"model":        func() map[string]any { return wrapModel(s.Model()) },
		"newSeries":    s.NewSeries,
		"setSeries":    s.SetSeries,
		"getSeries":    func(tag string) (map[string]any, error) { return itemJson(s.GetSeries(tag)) },
		"deleteSeries": s.DeleteSeries,
		"genSerie":     s.GenSerie,
		"genValue":     s.GenValue,
	}
}
