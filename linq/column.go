package linq

import (
	"regexp"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

// Type columns
type TypeColumn int

const (
	TpColumn TypeColumn = iota
	TpAtrib
	TpDetail
)

// String return string of type column
func (t TypeColumn) String() string {
	switch t {
	case TpColumn:
		return "Column"
	case TpAtrib:
		return "Atrib"
	case TpDetail:
		return "Detail"
	}
	return ""
}

type Required struct {
	Required bool
	Message  string
}

// Definition return a json with the definition of the required
func (r *Required) Definition() et.Json {
	return et.Json{
		"required": r.Required,
		"message":  r.Message,
	}
}

// Details is a function for details
type FuncDetail func(col *Column, data *et.Json)

// Validation tipe function
type Validation func(col *Column, value interface{}) bool

// Column is a struct for columns in a model
type Column struct {
	Model         *Model
	Name          string
	Tag           string
	Description   string
	TypeColumn    TypeColumn
	TypeData      TypeData
	Definition    et.Json
	Default       interface{}
	RelationTo    *Relation
	FuncDetail    FuncDetail
	Formula       string
	PrimaryKey    bool
	ForeignKey    bool
	Indexed       bool
	Unique        bool
	Hidden        bool
	IsSourceField bool
	Required      *Required
}

// name return a valid name of column, table, schema or database
func nAme(name string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_-]+")
	s := re.ReplaceAllString(name, "")
	s = strs.Replace(s, " ", "_")

	return strs.Uppcase(s)
}

func atribName(name string) string {
	name = nAme(name)

	return strs.Lowcase(name)
}

// IndexColumn return index of column in model
func IndexColumn(model *Model, name string) int {
	result := -1
	for i, col := range model.Columns {
		if strs.Uppcase(col.Name) == strs.Uppcase(name) {
			return i
		}
	}

	return result
}

// Column return a column in the model
func COlumn(model *Model, name string) *Column {
	idx := IndexColumn(model, name)
	if idx != -1 {
		return model.Columns[idx]
	}

	return nil
}

// NewColumn create a new column
func newColumn(model *Model, name, description string, typeColumm TypeColumn, typeData TypeData, _default interface{}) *Column {
	tag := strs.Lowcase(name)
	name = nAme(name)
	result := COlumn(model, name)
	if result != nil {
		return result
	}

	result = &Column{
		Model:         model,
		Name:          name,
		Tag:           tag,
		Description:   description,
		TypeColumn:    typeColumm,
		TypeData:      typeData,
		Definition:    *typeData.Definition(),
		Default:       _default,
		IsSourceField: name == SourceField.Up(),
	}

	if !model.UseStatus {
		model.UseStatus = result.Up() == StateField.Up()
	}

	if !model.UseSource {
		model.UseSource = result.Up() == SourceField.Up()
		if model.UseSource {
			model.Source = result
		}
	}

	if !model.UseCreatedTime {
		model.UseCreatedTime = TpCreatedTime == TpDate
	}

	if !model.UseCreatedBy {
		model.UseCreatedBy = TpCreatedBy == TpDate
	}

	if !model.UseLastEditedTime {
		model.UseLastEditedTime = TpLastEditedTime == TpDate
	}

	if !model.UseLastEditedBy {
		model.UseLastEditedBy = TpLastEditedBy == TpDate
	}

	if !model.UseProject {
		model.UseProject = TpProject == TpDate
	}

	model.AddColumn(result)

	if typeData.Indexed() {
		model.AddIndex(name, true)
	}

	return result
}

// Describe carapteristics of column
func (c *Column) DEfinition() et.Json {
	relationTo := et.Json{}
	if c.RelationTo != nil {
		relationTo = c.RelationTo.Definition()
	}

	required := et.Json{}
	if c.Required != nil {
		required = c.Required.Definition()
	}

	return et.Json{
		"schema":      c.Model.Schema.Name,
		"model":       c.Model.Name,
		"name":        c.Name,
		"tag":         c.Tag,
		"description": c.Description,
		"type_column": c.TypeColumn.String(),
		"type_data":   c.TypeData.String(),
		"definition":  c.Definition,
		"default":     c.Default,
		"relationTo":  relationTo,
		"formula":     c.Formula,
		"primaryKey":  c.PrimaryKey,
		"foreignKey":  c.ForeignKey,
		"indexed":     c.Indexed,
		"unique":      c.Unique,
		"hidden":      c.Hidden,
		"sourceField": c.IsSourceField,
		"required":    required,
	}
}

// AsModel return as name of model
func (c *Column) AsModel(l *Linq) string {
	f := l.GetFrom(c.Model)
	return f.AS
}

// AsModel return as name of model
func (c *Column) As(l *Linq) string {
	f := l.GetFrom(c.Model)
	s := l.GetColumn(c)
	if s.AS != c.Name {
		return strs.Format(`%s.%s AS %s`, f.AS, c.Name, s.AS)
	}

	return strs.Format(`%s.%s`, f.AS, c.Name)
}

// Table return table name of column
func (c *Column) Table() string {
	return c.Model.Table
}

// Return name of column in array string
func (c *Column) PrimaryKeys() []string {
	var result []string
	for _, v := range c.Model.PrimaryKeys {
		result = append(result, v.Name)
	}

	return result
}

// Hidden set hidden column
func (c *Column) SetHidden(val bool) {
	c.Hidden = val

	if val {
		c.Model.Hidden = append(c.Model.Hidden, c)
	}
}

func (c *Column) SetUnique(val bool) {
	if val {
		c.Model.AddUnique(c.Name, true)
	}
}

// SetRequired set required column
func (c *Column) SetRequired(val bool, msg string) {
	c.Required = &Required{
		Required: val,
		Message:  msg,
	}

	if val {
		c.Model.Required = append(c.Model.Required, c)
	}
}

// SetIndexed add a index to column
func (c *Column) SetIndexed(asc bool) {
	c.Model.AddIndex(c.Name, asc)
}

// SetRequiredTo set required column to model
func (c *Column) SetRelationTo(parent *Model, parentKey []string, _select []string, calculate TpCaculate, limit int) {
	c.RelationTo = &Relation{
		ForeignKey: []string{c.Name},
		Parent:     parent,
		ParentKey:  parentKey,
		Select:     _select,
		Calculate:  calculate,
		Limit:      limit,
	}
}

// Resutn name of column in uppercase
func (c *Column) Up() string {
	return strs.Uppcase(c.Name)
}

// Resutn name of column in lowercase
func (c *Column) Low() string {
	return strs.Lowcase(c.Name)
}
