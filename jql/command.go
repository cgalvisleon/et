package jql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type CommandType string

const (
	INSERT CommandType = "insert"
	UPDATE CommandType = "update"
	DELETE CommandType = "delete"
	UPSERT CommandType = "upsert"
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
func (c *Command) Where(cond *et.Condition) *Command {
	c.Conditions = append(c.Conditions, cond)
	return c
}

/**
* And
* @param cond *et.Condition
* @return *Command
**/
func (c *Command) And(cond *et.Condition) *Command {
	cond.Connector = et.And
	c.Conditions = append(c.Conditions, cond)
	return c
}

/**
* Or
* @param cond *et.Condition
* @return *Command
**/
func (c *Command) Or(cond *et.Condition) *Command {
	cond.Connector = et.Or
	c.Conditions = append(c.Conditions, cond)
	return c
}

/**
* BeforeInsert
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) BeforeInsert(fn TriggerFunction) *Command {
	c.beforeInserts = append(c.beforeInserts, fn)
	return c
}

/**
* BeforeUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) BeforeUpdate(fn TriggerFunction) *Command {
	c.beforeUpdates = append(c.beforeUpdates, fn)
	return c
}

/**
* BeforeDelete
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) BeforeDelete(fn TriggerFunction) *Command {
	c.beforeDeletes = append(c.beforeDeletes, fn)
	return c
}

/**
* BeforeInsertOrUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (c *Command) BeforeInsertOrUpdate(fn TriggerFunction) *Command {
	c.beforeInserts = append(c.beforeInserts, fn)
	c.beforeUpdates = append(c.beforeUpdates, fn)
	return c
}

/**
* AfterInsert
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) AfterInsert(fn TriggerFunction) *Command {
	c.afterInserts = append(c.afterInserts, fn)
	return c
}

/**
* AfterUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) AfterUpdate(fn TriggerFunction) *Command {
	c.afterUpdates = append(c.afterUpdates, fn)
	return c
}

/**
* AfterInsertOrUpdate
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) AfterInsertOrUpdate(fn TriggerFunction) *Command {
	c.afterInserts = append(c.afterInserts, fn)
	c.afterUpdates = append(c.afterUpdates, fn)
	return c
}

/**
* AfterDelete
* @param fn TriggerFunction
* @return *Command
**/
func (c *Command) AfterDelete(fn TriggerFunction) *Command {
	c.afterDeletes = append(c.afterDeletes, fn)
	return c
}
