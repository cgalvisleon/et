package jsql

import (
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
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
	Type          CommandType       `json:"type"`
	From          *From             `json:"from"`
	Data          []et.Json         `json:"data"`
	Conditions    []*et.Condition   `json:"conditions"`
	beforeInserts []TriggerFunction `json:"-"`
	beforeUpdates []TriggerFunction `json:"-"`
	beforeDeletes []TriggerFunction `json:"-"`
	afterInserts  []TriggerFunction `json:"-"`
	afterUpdates  []TriggerFunction `json:"-"`
	afterDeletes  []TriggerFunction `json:"-"`
}

/**
* newCommand
* @param model *Model, tp CommandType
* @return *Command
**/
func newCommand(model *Model, tp CommandType) *Command {
	return &Command{
		Type:          tp,
		From:          getFrom(model, ""),
		Data:          []et.Json{},
		Conditions:    []*et.Condition{},
		beforeInserts: []TriggerFunction{},
		beforeUpdates: []TriggerFunction{},
		beforeDeletes: []TriggerFunction{},
		afterInserts:  []TriggerFunction{},
		afterUpdates:  []TriggerFunction{},
		afterDeletes:  []TriggerFunction{},
	}
}

/**
* serialize
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
* ToJson
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
* Where
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Where(cond *et.Condition) *Command {
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* And
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) And(cond *et.Condition) *Command {
	cond.Connector = et.And
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* Or
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Or(cond *et.Condition) *Command {
	cond.Connector = et.Or
	s.Conditions = append(s.Conditions, cond)
	return s
}

/**
* BeforeInsert
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeInsert(fn TriggerFunction) *Command {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* BeforeUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeUpdate(fn TriggerFunction) *Command {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* BeforeDelete
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeDelete(fn TriggerFunction) *Command {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* BeforeInsertOrUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (s *Command) BeforeInsertOrUpdate(fn TriggerFunction) *Command {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterInsert
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsert(fn TriggerFunction) *Command {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* AfterUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterUpdate(fn TriggerFunction) *Command {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterInsertOrUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsertOrUpdate(fn TriggerFunction) *Command {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterDelete
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterDelete(fn TriggerFunction) *Command {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* insert
* @param tx *Tx
* @return (et.Items, error)
**/
func (s *Command) insert(tx *Tx) (et.Items, error) {
	if tx == nil {
		return et.Items{}, errors.New(msg.MSG_TRANSACTION_IS_NIL)
	}

	return et.Items{}, nil
}

/**
* update
* @param tx *Tx
* @return (et.Items, error)
**/
func (s *Command) update(tx *Tx) (et.Items, error) {
	if tx == nil {
		return et.Items{}, errors.New(msg.MSG_TRANSACTION_IS_NIL)
	}

	return et.Items{}, nil
}

/**
* delete
* @param tx *Tx
* @return (et.Items, error)
**/
func (s *Command) delete(tx *Tx) (et.Items, error) {
	if tx == nil {
		return et.Items{}, errors.New(msg.MSG_TRANSACTION_IS_NIL)
	}

	return et.Items{}, nil
}

/**
* upsert
* @param tx *Tx
* @return (et.Items, error)
**/
func (s *Command) upsert(tx *Tx) (et.Items, error) {
	if tx == nil {
		return et.Items{}, errors.New(msg.MSG_TRANSACTION_IS_NIL)
	}

	return et.Items{}, nil
}

/**
* ExecTx
* @param tx *Tx
* @return (et.Items, error)
**/
func (s *Command) ExecTx(tx *Tx) (et.Items, error) {
	tx, isCommitted := getTx(tx)
	switch s.Type {
	case INSERT:
		return s.insert(tx)
	case BULK:
		return s.insert(tx)
	case UPDATE:
		return s.update(tx)
	case DELETE:
		return s.delete(tx)
	case UPSERT:
		return s.upsert(tx)
	}

	if isCommitted {
		err := tx.commit()
		if err != nil {
			return et.Items{}, err
		}
	}

	return et.Items{}, nil
}
