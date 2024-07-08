package linq

import (
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

// TypeCommand struct to use in linq
type TypeCommand int

// Values for TypeCommand
const (
	Tpnone TypeCommand = iota
	TpInsert
	TpUpdate
	TpDelete
)

// String method to use in linq
func (d TypeCommand) String() string {
	switch d {
	case Tpnone:
		return "none"
	case TpInsert:
		return "insert"
	case TpUpdate:
		return "update"
	case TpDelete:
		return "delete"
	}
	return ""
}

type Value struct {
	Column *Column
	Old    interface{}
	New    interface{}
	Change bool
}

// Command struct to use in linq
type Values struct {
	Linq        *Linq
	From        *Lfrom
	TypeCommand TypeCommand
	Data        et.Json
	Values      []*Value
	Change      bool
	Ctid        string
	User        interface{}
	Project     interface{}
}

/**
* values returns a json with the values of the command
* @return et.Json
**/
func (l *Values) values() []et.Json {
	var result []et.Json
	for _, v := range l.Values {
		result = append(result, et.Json{
			"column": v.Column.Name,
			"old":    v.Old,
			"new":    v.New,
			"change": v.Change,
		})
	}

	return result
}

/**
* Definition returns a json with the definition of the command
* @return et.Json
**/
func (l *Values) Definition() et.Json {
	return et.Json{
		"from":        l.From.Definition(),
		"typeCommand": l.TypeCommand.String(),
		"data":        l.Data,
		"values":      l.values(),
		"change":      l.Change,
		"user":        l.User,
		"project":     l.Project,
	}
}

/**
* Set values to command
* @param col interface{}
* @param value interface{}
**/
func (l *Values) Set(col interface{}, value interface{}) {
	if col == nil {
		return
	}

	var _col *Column
	switch v := col.(type) {
	case string:
		_col = l.From.Column(v)
	case *Column:
		_col = v
	}

	if _col == nil {
		return
	}

	idx := slices.IndexFunc(l.Values, func(e *Value) bool { return e.Column == _col })
	if idx == -1 {
		l.Values = append(l.Values, &Value{
			Column: _col,
			Old:    value,
			New:    value,
		})
	} else {
		l.Values[idx].New = value
		l.Values[idx].Change = true
		if !l.Change {
			l.Change = true
		}
	}
}

/**
* Default return the default value of the column in command
* @param col *Column
* @return interface{}
**/
func (l *Values) Default(col *Column) interface{} {
	switch col.TypeData {
	case TpStatus:
		return col.TypeData.Default()
	case TpCreatedTime:
		return time.Now()
	case TpCreatedBy:
		return l.User
	case TpLastEditedTime:
		return time.Now()
	case TpLastEditedBy:
		return l.User
	case TpProject:
		return l.Project
	}

	return col.Default
}

/**
* newValues create a new values
* @param from *Lfrom
* @param tp TypeCommand
* @return *Values
**/
func newValues(from *Lfrom, tp TypeCommand) *Values {
	return &Values{
		From:        from,
		TypeCommand: tp,
		Data:        et.Json{},
		Values:      []*Value{},
		Change:      false,
		User:        "",
		Project:     "",
	}
}

/**
* consolidate values
**/
func (c *Values) consolidate(data et.Json) {
	if c.TypeCommand == Tpnone {
		return
	}

	from := c.From
	model := from.Model

	newAtrib := func(name string, value interface{}) *Column {
		var tp TypeData
		tp.Mutate(value)

		return model.DefineAtrib(name, "", tp, *tp.Definition())
	}

	properties := make(map[string]bool)
	if c.TypeCommand == TpInsert {
		for _, col := range model.Columns {
			if col.TypeColumn == TpDetail {
				continue
			}

			key := col.Low()
			def := c.Default(col)
			val := data.Get(key)
			if val == nil {
				val = def
			}
			c.Set(col, val)
			properties[key] = true
		}

		if model.Integrity {
			return
		}

		for k, v := range data {
			if properties[k] {
				continue
			}

			col := newAtrib(k, v)
			c.Set(col, v)
		}
	} else {
		for k, v := range c.Data {
			col := model.Column(k)
			if col.TypeData == TpData {
				continue
			} else if col == nil && model.Integrity {
				continue
			} else if col == nil {
				col = newAtrib(k, v)
			}

			old := data.Get(col.Low())
			c.Set(col, old)
			c.Set(col, v)
		}
	}
}

/**
* query, evaluate this model if use columnData and return the result for this condition
* @param sql string
* @param args ...any
* @return et.Items
**/
func (c *Values) query(sql string, args ...any) (et.Items, error) {
	var err error
	if c.From.Model.ColumnData == nil {
		items, err := c.Linq.query(sql, args...)
		if err != nil {
			return et.Items{}, err
		}

		return items, nil
	}

	items, err := c.Linq.querySource(sql, args...)
	if err != nil {
		return et.Items{}, err
	}

	return items, nil
}

/**
* curren, return the current values of the model
* @return et.Items
* @return error
**/
func (c *Values) curren() (et.Items, error) {
	currentSql, err := c.Linq.currentSql()
	if err != nil {
		return et.Items{}, err
	}

	result, err := c.query(currentSql)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* beforeInsert, execute before insert triggers
* @return error
**/
func (c *Values) beforeInsert() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeInsert {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterInsert, execute after insert triggers
* @return error
**/
func (c *Values) afterInsert() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterInsert {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* beforeUpdate, execute before update triggers
* @return error
**/
func (c *Values) beforeUpdate() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeUpdate {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterUpdate, execute after update triggers
* @return error
**/
func (c *Values) afterUpdate() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterUpdate {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* beforeDelete, execute before delete triggers
* @return error
**/
func (c *Values) beforeDelete() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeDelete {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterDelete, execute after delete triggers
* @return error
**/
func (c *Values) afterDelete() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterDelete {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Insert method to use in linq
* @return error
**/
func (c *Values) Insert() error {
	var err error
	c.consolidate(c.Data)

	err = c.beforeInsert()
	if err != nil {
		return err
	}

	c.Linq.Returns.Used = true
	c.Linq.Sql, err = c.Linq.insertSql()
	if err != nil {
		return err
	}

	items, err := c.query(c.Linq.Sql)
	if err != nil {
		return err
	}

	c.Linq.Result = &items
	for _, data := range items.Result {
		for k, v := range data {
			c.Set(k, v)
		}
	}

	err = c.afterInsert()
	if err != nil {
		return err
	}

	return nil
}

/**
* Update method to use in linq
* @return error
**/
func (c *Values) Update() error {
	current, err := c.curren()
	if err != nil {
		return err
	}

	if !current.Ok {
		return nil
	}

	if current.Count > MaxUpdate {
		return logs.Errorf("Update only allow %d items", MaxUpdate)
	}

	for _, old := range current.Result {
		ctid := old.Get("ctid")
		if ctid != nil {
			c.Ctid = ctid.(string)
		}

		c.consolidate(old)

		if !c.Change {
			continue
		}

		err = c.beforeUpdate()
		if err != nil {
			return err
		}

		c.Linq.Sql, err = c.Linq.updateSql()
		if err != nil {
			return err
		}

		items, err := c.query(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &items
		for _, data := range items.Result {
			for k, v := range data {
				c.Set(k, v)
			}
		}

		err = c.afterUpdate()
		if err != nil {
			return err
		}

		go c.UpdateCascade()
	}

	return nil
}

/**
* Delete method to use in linq
* @return error
**/
func (c *Values) Delete() error {
	current, err := c.curren()
	if err != nil {
		return err
	}

	if !current.Ok {
		return nil
	}

	if current.Count > MaxDelete {
		return logs.Errorf("Delete only allow %d items", MaxDelete)
	}

	for _, old := range current.Result {
		ctid := old.Get("ctid")
		if ctid != nil {
			c.Ctid = ctid.(string)
		}

		c.consolidate(old)

		err = c.beforeDelete()
		if err != nil {
			return err
		}

		c.Linq.Sql, err = c.Linq.deleteSql()
		if err != nil {
			return err
		}

		items, err := c.query(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &items

		err = c.afterDelete()
		if err != nil {
			return err
		}

		go c.DeleteCascade()
	}

	return nil
}

/**
* UpdateCascade method to use in linq
* @return error
**/
func (c *Values) UpdateCascade() error {
	return nil
}

/**
* DeleteCascade method to use in linq
* @return error
**/
func (c *Values) DeleteCascade() error {
	return nil
}

/**
* Insert method to use in linq
* @return error
**/
func (m *Model) Insert(data et.Json) *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Values.From = l.Froms[0]
	l.Values.TypeCommand = TpInsert
	l.Values.Data = data

	return l
}

/**
* Update method to use in linq
* @return error
**/
func (m *Model) Update(data et.Json) *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Values.From = l.Froms[0]
	l.Values.TypeCommand = TpUpdate
	l.Values.Data = data

	return l
}

/**
* Delete method to use in linq
* @return error
**/
func (m *Model) Delete() *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Values.From = l.Froms[0]
	l.Values.TypeCommand = TpUpdate

	return l
}
