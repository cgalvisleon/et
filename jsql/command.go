package jsql

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
)

type CommandType string

const (
	INSERT CommandType = "insert"
	UPDATE CommandType = "update"
	DELETE CommandType = "delete"
	UPSERT CommandType = "upsert"
	BULK   CommandType = "bulk"
)

type Command struct {
	ID             string            `json:"id"`
	Type           CommandType       `json:"type"`
	From           *F                `json:"from"`
	Data           []et.Json         `json:"data"`
	New            et.Json           `json:"new"`
	Old            et.Json           `json:"old"`
	Conditions     []*et.Condition   `json:"conditions"`
	Returns        []string          `json:"returns"`
	UseSourceField bool              `json:"use_source_field"`
	BeforeInserts  []*jrex.Jrex      `json:"before_inserts"`
	BeforeUpdates  []*jrex.Jrex      `json:"before_updates"`
	BeforeDeletes  []*jrex.Jrex      `json:"before_deletes"`
	AfterInserts   []*jrex.Jrex      `json:"after_inserts"`
	AfterUpdates   []*jrex.Jrex      `json:"after_updates"`
	AfterDeletes   []*jrex.Jrex      `json:"after_deletes"`
	beforeInserts  []TriggerFunction `json:"-"`
	beforeUpdates  []TriggerFunction `json:"-"`
	beforeDeletes  []TriggerFunction `json:"-"`
	afterInserts   []TriggerFunction `json:"-"`
	afterUpdates   []TriggerFunction `json:"-"`
	afterDeletes   []TriggerFunction `json:"-"`
	db             *DB               `json:"-"`
	model          *Model            `json:"-"`
	isDebug        bool              `json:"-"`
	isTest         bool              `json:"-"`
}

/**
* newCommand: Constructs a Command of the given type, copying the model's trigger slices.
* @param model *Model, tp CommandType
* @return *Command
**/
func newCommand(model *Model, tp CommandType) *Command {
	result := &Command{
		ID:             reg.GenULID("command"),
		Type:           tp,
		From:           getFrom(model, ""),
		Data:           []et.Json{},
		New:            et.Json{},
		Old:            et.Json{},
		Conditions:     []*et.Condition{},
		Returns:        []string{},
		UseSourceField: model.SourceField != "",
		BeforeInserts:  make([]*jrex.Jrex, 0),
		BeforeUpdates:  make([]*jrex.Jrex, 0),
		BeforeDeletes:  make([]*jrex.Jrex, 0),
		AfterInserts:   make([]*jrex.Jrex, 0),
		AfterUpdates:   make([]*jrex.Jrex, 0),
		AfterDeletes:   make([]*jrex.Jrex, 0),
		beforeInserts:  []TriggerFunction{},
		beforeUpdates:  []TriggerFunction{},
		beforeDeletes:  []TriggerFunction{},
		afterInserts:   []TriggerFunction{},
		afterUpdates:   []TriggerFunction{},
		afterDeletes:   []TriggerFunction{},
		db:             model.db,
		model:          model,
		isDebug:        model.db.IsDebug,
	}
	if map[CommandType]bool{INSERT: true, BULK: true, UPSERT: true}[tp] {
		for _, fn := range model.beforeInserts {
			result.beforeInserts = append(result.beforeInserts, fn)
		}
		for _, fn := range model.afterInserts {
			result.afterInserts = append(result.afterInserts, fn)
		}
		for _, code := range model.BeforeInserts {
			result.BeforeInserts = append(result.BeforeInserts, code)
		}
		for _, code := range model.AfterInserts {
			result.AfterInserts = append(result.AfterInserts, code)
		}
	}
	if map[CommandType]bool{UPDATE: true, UPSERT: true}[tp] {
		for _, fn := range model.beforeUpdates {
			result.beforeUpdates = append(result.beforeUpdates, fn)
		}
		for _, fn := range model.afterUpdates {
			result.afterUpdates = append(result.afterUpdates, fn)
		}
		for _, code := range model.BeforeUpdates {
			result.BeforeUpdates = append(result.BeforeUpdates, code)
		}
		for _, code := range model.AfterUpdates {
			result.AfterUpdates = append(result.AfterUpdates, code)
		}
	}
	if map[CommandType]bool{DELETE: true}[tp] {
		for _, fn := range model.beforeDeletes {
			result.beforeDeletes = append(result.beforeDeletes, fn)
		}
		for _, fn := range model.afterDeletes {
			result.afterDeletes = append(result.afterDeletes, fn)
		}
		for _, code := range model.BeforeDeletes {
			result.BeforeDeletes = append(result.BeforeDeletes, code)
		}
		for _, code := range model.AfterDeletes {
			result.AfterDeletes = append(result.AfterDeletes, code)
		}
	}
	return result
}

/**
* serialize: Marshals the command metadata to JSON bytes.
* @return []byte, error
**/
func (s *Command) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson: Returns the command metadata as an et.Json map.
* @return et.Json
**/
func (s *Command) ToJson() et.Json {
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
* setDebug: Sets the debug flag for the query.
* @param debug bool
* @return *Query
**/
func (s *Command) setDebug(debug bool) *Command {
	s.isDebug = debug
	return s
}

/**
* Debug: Enables debug mode — SQL is logged to stdout.
* @return *Command
**/
func (s *Command) Debug() *Command {
	return s.setDebug(true)
}

/**
* Test: Enables test mode — SQL is generated but not executed.
* @return *Command
**/
func (s *Command) Test() *Command {
	s.isTest = true
	return s
}

/**
* addCondition: Appends a condition to the command's WHERE clause list.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) addCondition(cond *et.Condition) *Command {
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* Where: Sets the first WHERE condition and returns the command for chaining.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Where(cond *et.Condition) *Command {
	return s.addCondition(cond)
}

/**
* And: Appends a condition joined with AND to the WHERE clause.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) And(cond *et.Condition) *Command {
	cond.Connector = et.And
	return s.addCondition(cond)
}

/**
* Or: Appends a condition joined with OR to the WHERE clause.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Or(cond *et.Condition) *Command {
	cond.Connector = et.Or
	return s.addCondition(cond)
}

/**
* Return: Sets the fields to return in the result.
* @param fields ...string
* @return *Command
**/
func (s *Command) Return(fields ...string) *Command {
	s.Returns = fields
	return s
}

/**
* BeforeInsert: Registers a trigger function to run before each INSERT execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeInsert(fn TriggerFunction) *Command {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* BeforeUpdate: Registers a trigger function to run before each UPDATE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeUpdate(fn TriggerFunction) *Command {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* BeforeDelete: Registers a trigger function to run before each DELETE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeDelete(fn TriggerFunction) *Command {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* BeforeInsertOrUpdate: Registers a trigger function to run before INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeInsertOrUpdate(fn TriggerFunction) *Command {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterInsert: Registers a trigger function to run after each INSERT execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsert(fn TriggerFunction) *Command {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* AfterUpdate: Registers a trigger function to run after each UPDATE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterUpdate(fn TriggerFunction) *Command {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterInsertOrUpdate: Registers a trigger function to run after INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsertOrUpdate(fn TriggerFunction) *Command {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterDelete: Registers a trigger function to run after each DELETE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterDelete(fn TriggerFunction) *Command {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* insert: Executes INSERT for each row in Data, running before/after triggers per row.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) insert(tx *Tx) (et.Items, error) {
	if len(s.Data) == 0 {
		return et.Items{}, nil
	}

	result := et.NewItems([]et.Json{})
	items := s.Data
	model := s.model
	for _, new := range items {
		s.New = new

		for _, col := range model.Required {
			if _, ok := new[col.Name]; !ok {
				return et.Items{}, fmt.Errorf(MSG_REQUIRED_FIELD, col.Name)
			}
		}

		for _, tg := range s.beforeInserts {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.BeforeInserts {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		sql, err := s.db.command(s)
		if err != nil {
			return et.Items{}, err
		}

		if s.isDebug {
			logs.Debug("INSERT:", sql)
		}

		if !s.isTest {
			_, err = s.db.SqlTx(tx, sql)
			if err != nil {
				return et.Items{}, err
			}
		}

		for _, tg := range s.afterInserts {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.AfterInserts {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		result.Add(s.New)
	}

	return result, nil
}

/**
* update: Fetches matching rows, merges Data[0] into each, and executes UPDATE with triggers.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) update(tx *Tx) (et.Items, error) {
	if len(s.Data) == 0 {
		return et.Items{}, nil
	}

	result := et.NewItems([]et.Json{})
	model := s.model
	current, err := newQuery(model).
		addCondition(s.Conditions).
		All()
	if err != nil {
		return et.Items{}, err
	}

	data := s.Data[0]
	for _, old := range current.Result {
		s.Old = old
		s.New = s.Old.Clone()
		maps.Copy(s.New, data)
		for _, tg := range s.beforeUpdates {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.BeforeUpdates {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		sql, err := s.db.command(s)
		if err != nil {
			return et.Items{}, err
		}

		if s.isDebug {
			logs.Debug("UPDATE:", sql)
		}

		if !s.isTest {
			_, err = s.db.SqlTx(tx, sql)
			if err != nil {
				return et.Items{}, err
			}
		}

		for _, tg := range s.afterUpdates {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.AfterUpdates {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		result.Add(s.New)
	}

	return result, nil
}

/**
* delete: Fetches matching rows and executes DELETE for each with before/after triggers.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) delete(tx *Tx) (et.Items, error) {
	result := et.NewItems([]et.Json{})
	model := s.model
	items, err := newQuery(model).
		addCondition(s.Conditions).
		All()
	if err != nil {
		return et.Items{}, err
	}

	data := s.Data[0]
	for _, old := range items.Result {
		s.Old = old
		s.New = et.Json{}
		maps.Copy(s.New, data)
		for _, tg := range s.beforeDeletes {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.BeforeDeletes {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		sql, err := s.db.command(s)
		if err != nil {
			return et.Items{}, err
		}

		if s.isDebug {
			logs.Debug("DELETE:", sql)
		}

		if !s.isTest {
			_, err = s.db.SqlTx(tx, sql)
			if err != nil {
				return et.Items{}, err
			}
		}

		for _, tg := range s.afterDeletes {
			if err := tg(tx, s.Old, s.New); err != nil {
				return et.Items{}, err
			}
		}

		for _, jrex := range s.AfterDeletes {
			jrex.Set("old", s.Old)
			jrex.Set("new", s.New)
			_, err := jrex.Run()
			if err != nil {
				return et.Items{}, err
			}
			s.Old = jrex.GetJson("old")
			s.New = jrex.GetJson("new")
		}

		result.Add(s.Old)
	}

	return result, nil
}

/**
* upsert: Resolves to INSERT when no row matches the conditions, or UPDATE when exactly one does.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) upsert(tx *Tx) (et.Items, error) {
	model := s.model
	isExists, err := newQuery(model).
		addCondition(s.Conditions).
		setDebug(s.isDebug).
		ExistsTx(tx)
	if err != nil {
		return et.Items{}, err
	}

	if isExists {
		s.Type = UPDATE
		return s.update(tx)
	}

	s.Type = INSERT
	return s.insert(tx)
}

/**
* ExecTx: Dispatches the command to the appropriate handler and commits if no external Tx was given.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) ExecTx(tx *Tx) (et.Items, error) {
	var err error
	var result et.Items
	tx, isCommitted := getTx(tx)
	switch s.Type {
	case INSERT:
		result, err = s.insert(tx)
	case BULK:
		result, err = s.insert(tx)
	case UPDATE:
		result, err = s.update(tx)
	case DELETE:
		result, err = s.delete(tx)
	case UPSERT:
		result, err = s.upsert(tx)
	}
	if err != nil {
		return et.Items{}, err
	}

	if isCommitted {
		err = tx.commit()
		if err != nil {
			return et.Items{}, err
		}
	}

	return result, nil
}

/**
* Exec: Executes the command without an explicit transaction.
* @return et.Items, error
**/
func (s *Command) Exec() (et.Items, error) {
	return s.ExecTx(nil)
}

/**
* OneTx: Executes the command and returns the first result within the given transaction.
* @param tx *Tx
* @return et.Item, error
**/
func (s *Command) OneTx(tx *Tx) (et.Item, error) {
	items, err := s.ExecTx(tx)
	if err != nil {
		return et.Item{}, err
	}

	return items.First()
}

/**
* One: Executes the command and returns the first result without an explicit transaction.
* @return et.Item, error
**/
func (s *Command) One() (et.Item, error) {
	return s.OneTx(nil)
}

/**
* loadQuery: Loads a query from a JSON object.
* @param tx *Tx, query et.Json
* @return et.Items, error
**/
func (s *Command) loadQuery(tx *Tx, query et.Json) (et.Items, error) {
	s.Conditions = et.ToCondition(query)
	s.Returns = query.ArrayStr("returns")
	return s.ExecTx(tx)
}
