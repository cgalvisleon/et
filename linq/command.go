package linq

import (
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
	TpUdsert
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
	case TpUdsert:
		return "upsert"
	case TpDelete:
		return "delete"
	}
	return ""
}

// Command struct to use in linq
type Lcommand struct {
	Linq        *Linq
	From        *Lfrom
	TypeCommand TypeCommand
	Data        *et.Json
	Old         *et.Json
	New         *et.Json
	Change      *et.Json
	Columns     *et.Json
	Atribs      *et.Json
	User        et.Json
	Project     et.Json
}

// Definition method to use in linq
func (l *Lcommand) Definition() et.Json {
	return et.Json{
		"from":        l.From.Definition(),
		"typeCommand": l.TypeCommand.String(),
		"data":        l.Data,
		"columns":     l.Columns,
		"atrib":       l.Atribs,
		"old":         l.Old,
		"new":         l.New,
		"change":      l.Change,
	}
}

// Return default value for column
func (l *Lcommand) Default(col *Column) interface{} {
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

// NewCommand method to use in linq
func newCommand(from *Lfrom, tp TypeCommand) *Lcommand {
	return &Lcommand{
		From:        from,
		TypeCommand: tp,
		Data:        &et.Json{},
		Old:         &et.Json{},
		New:         &et.Json{},
		Change:      &et.Json{},
		Columns:     &et.Json{},
		Atribs:      &et.Json{},
	}
}

// Add key value to command source
func (c *Lcommand) setSource(col *Column, value interface{}) {
	if col.TypeColumn == TpDetail {
		return
	}

	if col.TypeColumn == TpColumn {
		c.Columns.Set(col.Low(), value)
	}

	if col.TypeColumn == TpAtrib {
		c.Atribs.Set(col.Low(), value)
	}
}

// Add key value to command new
func (c *Lcommand) setNew(key string, value interface{}) {
	if c.New == nil {
		c.New = &et.Json{}
	}

	if c.New.Get(key) == nil {
		c.New.Set(key, value)
	}
}

// Consolidate data to command new
func (c *Lcommand) consolidate() {
	if c.TypeCommand == Tpnone {
		return
	}

	if c.TypeCommand == TpDelete {
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
			val := c.Data.Get(key)
			if val == nil {
				val = def
			}
			c.setSource(col, val)
			c.setNew(key, val)
			properties[key] = true
		}

		if model.Integrity {
			return
		}

		for k, v := range *c.Data {
			if properties[k] {
				continue
			}

			col := newAtrib(k, v)
			c.setSource(col, v)
			c.setNew(k, v)
		}
	} else {
		for k, v := range *c.Data {
			col := model.Column(k)
			if col == nil && model.Integrity {
				continue
			} else if col == nil {
				col = newAtrib(k, v)
			}

			c.setSource(col, v)
			c.setNew(k, v)
		}
	}
}

// Query method to use in linq
func (c *Lcommand) query(sql string, args ...any) (et.Items, error) {
	var items et.Items
	var err error
	if c.From.Model.UseSource {
		items, err = c.Linq.querySource(c.Linq.Sql, args...)
		if err != nil {
			return et.Items{}, err
		}
	} else {
		items, err = c.Linq.query(c.Linq.Sql, args...)
		if err != nil {
			return et.Items{}, err
		}
	}

	return items, nil
}

// Get current values
func (c *Lcommand) curren() (et.Items, error) {
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

// Execute before insert triggers
func (c *Lcommand) beforeInsert() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeInsert {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute after insert triggers
func (c *Lcommand) afterInsert() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterInsert {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute before update triggers
func (c *Lcommand) beforeUpdate() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeUpdate {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute after update triggers
func (c *Lcommand) afterUpdate() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterUpdate {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute before delete triggers
func (c *Lcommand) beforeDelete() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.BeforeDelete {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute after delete triggers
func (c *Lcommand) afterDelete() error {
	f := c.From
	m := f.Model

	for _, trigger := range m.AfterDelete {
		err := trigger(m, c.Old, c.New, *c.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute insert function
func (c *Lcommand) Insert() error {
	var err error
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
	if items.Ok {
		c.New = &items.Result[0]
	}

	err = c.afterInsert()
	if err != nil {
		return err
	}

	return nil
}

// Execute update function
func (c *Lcommand) update(current et.Items) error {
	if current.Count > MaxUpdate {
		return logs.Errorf("Update only allow %d items", MaxUpdate)
	}

	var err error
	form := c.From
	model := form.Model
	ch := false
	new := *c.New
	for _, data := range current.Result {
		c.Old = &data
		c.New, ch = data.Merge(new)
		c.Change, _ = data.Chage(new)

		if !ch {
			continue
		}

		err = c.beforeUpdate()
		if err != nil {
			return err
		}

		_idt := c.Old.Get(IdTField.Low())
		if _idt == nil {
			return logs.Errorm("No idT in data")
		}

		c.Linq.Returns.Used = true
		c.Linq.Where(model.C(IdTField.Low()).Eq(_idt))
		c.Linq.Sql, err = c.Linq.updateSql()
		if err != nil {
			return err
		}

		items, err := c.query(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &items
		if items.Ok {
			c.New = &items.Result[0]
		}

		go c.UpdateCascade()

		err = c.afterUpdate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Execute update function
func (c *Lcommand) Update() error {
	current, err := c.curren()
	if err != nil {
		return err
	}

	if !current.Ok {
		return nil
	}

	return c.update(current)
}

// Execute update or insert function
func (c *Lcommand) Upsert() error {
	current, err := c.curren()
	if err != nil {
		return err
	}

	if !current.Ok {
		return c.Insert()
	}

	return c.update(current)
}

// Execute delete function
func (c *Lcommand) Delete() error {
	current, err := c.curren()
	if err != nil {
		return err
	}

	if !current.Ok {
		return nil
	}

	if current.Count > MaxDelete {
		return logs.Errorf("Update only allow %d items", MaxDelete)
	}

	form := c.From
	model := form.Model
	for _, data := range current.Result {
		c.Old = &data

		err = c.beforeDelete()
		if err != nil {
			return err
		}

		_idt := c.Old.Get(IdTField.Low())
		if _idt == nil {
			return logs.Errorm("No idT in data")
		}

		c.Linq.Returns.Used = false
		c.Linq.Where(model.C(IdTField.Low()).Eq(_idt))
		c.Linq.Sql, err = c.Linq.deleteSql()
		if err != nil {
			return err
		}

		items, err := c.query(c.Linq.Sql)
		if err != nil {
			return err
		}

		c.Linq.Result = &items

		go c.DeleteCascade()

		err = c.afterDelete()
		if err != nil {
			return err
		}
	}

	return nil
}

// Update cascade data to linq
func (c *Lcommand) UpdateCascade() error {
	return nil
}

// Delete cascade data to linq
func (c *Lcommand) DeleteCascade() error {
	return nil
}

// Insert method to use in linq
func (m *Model) Insert(data et.Json) *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Command.From = l.Froms[0]
	l.Command.TypeCommand = TpInsert
	l.Command.Data = &data
	l.Command.consolidate()

	return l
}

// Update method to use in linq
func (m *Model) Update(data et.Json) *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Command.From = l.Froms[0]
	l.Command.TypeCommand = TpUpdate
	l.Command.Data = &data
	l.Command.consolidate()

	return l
}

// Update or insert method to use in linq
func (m *Model) Upsert(data et.Json) *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Command.From = l.Froms[0]
	l.Command.TypeCommand = TpUdsert
	l.Command.Data = &data
	l.Command.consolidate()

	return l
}

// Delete method to use in linq
func (m *Model) Delete() *Linq {
	l := From(m)
	l.TypeQuery = TpCommand
	l.Command.From = l.Froms[0]
	l.Command.TypeCommand = TpDelete
	l.Command.consolidate()

	return l
}
