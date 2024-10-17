package linq

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
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
	Model       *Model
	TypeCommand TypeCommand
	Data        et.Json
	Values      []*Value
	Change      bool
	History     bool
	IdT         string
	User        interface{}
	Project     interface{}
}

/**
* Old returns a json with the old values of the command
* @return et.Json
**/
func (v *Values) Olds() et.Json {
	result := et.Json{}
	for _, value := range v.Values {
		result[value.Column.Low()] = value.Old
	}

	return result
}

/**
* New returns a json with the new values of the command
* @return et.Json
**/
func (v *Values) News() et.Json {
	result := et.Json{}
	for _, value := range v.Values {
		result[value.Column.Low()] = value.New
	}

	return result
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
* Describe returns a json with the definition of the command
* @return et.Json
**/
func (l *Values) Describe() et.Json {
	return et.Json{
		"Model":       l.Model.Describe(),
		"typeCommand": l.TypeCommand.String(),
		"data":        l.Data,
		"values":      l.values(),
		"change":      l.Change,
		"idt":         l.IdT,
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
		_col = l.Model.Column(v)
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
* Old return the old value of the column in command
* @param def interface{}
* @param name string
* @return interface{}
**/
func (l *Values) Old(def interface{}, name string) interface{} {
	idx := slices.IndexFunc(l.Values, func(e *Value) bool { return e.Column.Low() == strs.Lowcase(name) })
	if idx == -1 {
		return def
	}

	return l.Values[idx].Old
}

func (l *Values) IsDifferent(name string) bool {
	old := l.Old(nil, name)
	new := l.New(nil, name)
	result := old != nil && new != nil && old != new
	return result
}

/**
* New return the new value of the column in command
* @param def interface{}
* @param name string
* @return interface{}
**/
func (l *Values) New(def interface{}, name string) interface{} {
	idx := slices.IndexFunc(l.Values, func(e *Value) bool { return e.Column.Low() == strs.Lowcase(name) })
	if idx == -1 {
		return def
	}

	return l.Values[idx].New
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
		return timezone.NowTime()
	case TpCreatedBy:
		return l.User
	case TpLastEditedTime:
		return timezone.NowTime()
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
		Model:       from.Model,
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
func (c *Values) consolidate(old, new et.Json) {
	if c.TypeCommand == Tpnone {
		return
	}

	model := c.Model
	newAtrib := func(name string, value interface{}) *Column {
		var tp TypeData
		tp.Mutate(value)

		return model.DefineAtrib(name, "", tp, *tp.Describe())
	}

	if c.TypeCommand == TpInsert {
		for _, col := range model.Columns {
			if col.TypeColumn > TpAtrib {
				continue
			}

			key := col.Low()
			def := c.Default(col)
			val := new.Get(key)
			if val == nil {
				val = def
			} else {
				delete(new, key)
			}
			c.Set(col, val)
		}

		if model.ColumnSource == nil {
			return
		}

		if model.Integrity {
			return
		}

		for k, v := range new {
			col := model.Col(k)
			if col == nil {
				col := newAtrib(k, v)
				c.Set(col, v)
			}
		}
	} else {
		for k, v := range new {
			col := model.Col(k)
			if col == nil && model.Integrity {
				continue
			} else if col == nil {
				col = newAtrib(k, v)
			} else if col.TypeData == TpSource {
				continue
			}

			old_val := old.Get(col.Low())
			c.Set(col, old_val)
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
	result, err := c.Linq.query(sql, args...)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* data, execute a query in the database
* @param sql string
* @param args ...any
* @return et.Items
* @return error
**/
func (c *Values) data(sql string, args ...any) (et.Items, error) {
	result, err := c.Linq.data(sql, args...)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* query, evaluate this model if use columnData and return the result for this condition
* @param sql string
* @param args ...any
* @return et.Item
**/
func (c *Values) command(sql string, args ...any) (et.Item, error) {
	item, err := c.Linq.command(sql, args...)
	if err != nil {
		return et.Item{}, err
	}

	return item, nil
}

/**
* curren, return the current values of the model
* @return et.Items
* @return error
**/
func (c *Values) curren() (et.Items, error) {
	sql, err := c.Linq.currentSql()
	if err != nil {
		return et.Items{}, err
	}

	if c.Model.ColumnSource == nil {
		result, err := c.query(sql)
		if err != nil {
			return et.Items{}, err
		}

		return result, nil
	}

	result, err := c.data(sql)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* beforeInserts, execute before insert triggers
* @return error
**/
func (c *Values) beforeInserts() error {
	m := c.Model

	for _, trigger := range m.BeforeInsert {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterInserts, execute after insert triggers
* @return error
**/
func (c *Values) afterInserts() error {
	m := c.Model

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
func (c *Values) beforeUpdates() error {
	m := c.Model

	for _, trigger := range m.BeforeUpdate {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterUpdates, execute after update triggers
* @return error
**/
func (c *Values) afterUpdates() error {
	m := c.Model

	for _, trigger := range m.AfterUpdate {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* beforeDeletes, execute before delete triggers
* @return error
**/
func (c *Values) beforeDeletes() error {
	m := c.Model

	for _, trigger := range m.BeforeDelete {
		err := trigger(m, c)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* afterDeletes, execute after delete triggers
* @return error
**/
func (c *Values) afterDeletes() error {
	m := c.Model

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
	c.consolidate(nil, c.Data)

	err = c.beforeInserts()
	if err != nil {
		return err
	}

	c.Linq.Sql, err = c.Linq.insertSql()
	if err != nil {
		return err
	}

	result, err := c.command(c.Linq.Sql)
	if err != nil {
		return err
	}

	c.Linq.Result = &result
	for k, v := range result.Result {
		c.Set(k, v)
	}

	err = c.afterInserts()
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
		return logs.Errorf(MSG_UPDATE_ONLY_ALLOW, MaxUpdate)
	}

	for _, old := range current.Result {
		idT := old.Key("_idt")
		if idT == "-1" {
			return logs.Errorm(MSG_NOT_FOUND_IDT)
		}

		c.consolidate(old, c.Data)

		if !c.Change {
			continue
		}

		err = c.beforeUpdates()
		if err != nil {
			return err
		}

		c.Linq.Sql, err = c.Linq.updateSql()
		if err != nil {
			return err
		}

		result, err := c.command(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &result
		for k, v := range result.Result {
			c.Set(k, v)
		}

		err = c.afterUpdates()
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
		return logs.Errorf(MSG_DELETE_ONLY_ALLOW, MaxDelete)
	}

	for _, old := range current.Result {
		idT := old.Key("_idt")
		if idT == "-1" {
			return logs.Errorm(MSG_NOT_FOUND_IDT)
		}

		c.consolidate(old, c.Data)

		err = c.beforeDeletes()
		if err != nil {
			return err
		}

		c.Linq.Sql, err = c.Linq.deleteSql()
		if err != nil {
			return err
		}

		result, err := c.command(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &result

		err = c.afterDeletes()
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
	l.Values.Model = l.Froms[0].Model
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
	l.Values.Model = l.Froms[0].Model
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
	l.Values.Model = l.Froms[0].Model
	l.Values.TypeCommand = TpDelete
	l.Values.Data = et.Json{}

	return l
}

/**
* History method to use in linq
* @return *Linq
**/
func (l *Linq) History(val bool) *Linq {
	l.Values.History = val
	return l
}
