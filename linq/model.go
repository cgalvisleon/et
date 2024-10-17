package linq

import (
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

/**
* TypeTrigger is a enum for trigger type
**/
type TypeTrigger int

const (
	BeforeInsert TypeTrigger = iota
	AfterInsert
	BeforeUpdate
	AfterUpdate
	BeforeDelete
	AfterDelete
)

/**
* String return string of type trigger
* @return string
**/
func (t TypeTrigger) String() string {
	switch t {
	case BeforeInsert:
		return "beforeInsert"
	case AfterInsert:
		return "afterInsert"
	case BeforeUpdate:
		return "beforeUpdate"
	case AfterUpdate:
		return "afterUpdate"
	case BeforeDelete:
		return "beforeDelete"
	case AfterDelete:
		return "afterDelete"
	}
	return ""
}

/**
* Constraint is a struct for constraint
**/
type Constraint struct {
	ForeignKey []string
	Parent     *Model
	ParentKey  []string
}

/**
* Describe return a json with the definition of the constraint
* @return et.Json
**/
func (c *Constraint) Describe() et.Json {
	return et.Json{
		"foreignKey": c.ForeignKey,
		"parent":     c.Parent.Name,
		"parentKey":  c.ParentKey,
	}
}

/**
* Fkey return a valid name of constraint
* @return string
**/
func (c *Constraint) Fkey() string {
	return strings.Join(c.ForeignKey, "_")
}

/**
* Table return a valid parent table name of constraint
* @return string
**/
func (c *Constraint) Table() string {
	return c.Parent.Table
}

/**
* Pkey return a valid parent key name of constraint
* @return string
**/
func (c *Constraint) Pkey() string {
	return strings.Join(c.ParentKey, "_")
}

/**
* Index is a struct for index
**/
type Index struct {
	Column *Column
	Asc    bool
}

/**
* Describe return a json with the definition of the index
* @return et.Json
**/
func (i *Index) Describe() et.Json {
	return et.Json{
		"column": i.Column.Name,
		"asc":    i.Asc,
	}
}

/**
* Trigger is a function for trigger
* @param model *Model
* @param old et.Json
* @param new et.Json
* @param data et.Json
* @return error
**/
type Trigger func(model *Model, value *Values) error

/**
* Listener is a function for listener
* @param data et.Json
**/
type Listener func(data et.Json)

/**
* RelationTo is a struct for relation to
**/
type RelationTo struct {
	PrimaryKey *Column
	ForeignKey *Column
}

/**
* Describe return a json with the definition of the relation to
* @return et.Json
**/
func (r *RelationTo) Describe() et.Json {
	return et.Json{
		"primaryKey": r.PrimaryKey.Name,
		"foreignKey": r.ForeignKey.Name,
	}
}

type DDL struct {
	Table       string
	Indexes     string
	ForeignKeys string
	Objects     string
	Recycling   string
}

/**
* Model is a struct for model
**/
type Model struct {
	Schema               *Schema
	DB                   *DB
	Name                 string
	Tag                  string
	Table                string
	Description          string
	Columns              []*Column
	PrimaryKeys          []*Column
	ForeignKey           []*Constraint
	Index                []*Index
	Unique               []*Index
	RelationTo           []*Column
	Details              []*Column
	Hidden               []*Column
	Required             []*Column
	ColumnStatus         *Column
	ColumnSource         *Column
	ColumnCreatedTime    *Column
	ColumnUpdatedTime    *Column
	ColumnCreatedBy      *Column
	ColumnLastEditedTime *Column
	ColumnLastEditedBy   *Column
	ColumnProject        *Column
	ColumnSerie          *Column
	ColumnUUIndex        *Column
	BeforeInsert         []Trigger
	AfterInsert          []Trigger
	BeforeUpdate         []Trigger
	AfterUpdate          []Trigger
	BeforeDelete         []Trigger
	AfterDelete          []Trigger
	OnListener           Listener
	Integrity            bool
	DDL                  *DDL
	Version              int
}

/**
* NewModel create a new model
* @param schema *Schema
* @param name string
* @param description string
* @param version int
* @return *Model
**/
func NewModel(schema *Schema, name, description string, version int) *Model {
	tag := strs.Lowcase(name)
	name = nameCase(name)
	table := strs.Append(schema.Name, name, ".")

	for _, v := range models {
		if strs.Uppcase(v.Table) == strs.Uppcase(table) {
			return v
		}
	}

	result := &Model{
		Schema:       schema,
		Name:         name,
		Tag:          tag,
		Table:        table,
		Description:  description,
		Columns:      []*Column{},
		PrimaryKeys:  []*Column{},
		ForeignKey:   []*Constraint{},
		Index:        []*Index{},
		Unique:       []*Index{},
		RelationTo:   []*Column{},
		Details:      []*Column{},
		Hidden:       []*Column{},
		Required:     []*Column{},
		BeforeInsert: []Trigger{},
		AfterInsert:  []Trigger{},
		BeforeUpdate: []Trigger{},
		AfterUpdate:  []Trigger{},
		BeforeDelete: []Trigger{},
		AfterDelete:  []Trigger{},
		OnListener:   nil,
		Integrity:    false,
		DDL:          &DDL{},
		Version:      version,
	}

	schema.AddModel(result)

	return result
}

/**
* Describe return a json with the definition of the model
* @return et.Json
**/
func (m *Model) Describe() et.Json {
	var columns []et.Json = []et.Json{}
	for _, v := range m.Columns {
		columns = append(columns, v.Describe())
	}

	var primaryKeys []string = []string{}
	for _, v := range m.PrimaryKeys {
		primaryKeys = append(primaryKeys, v.Name)
	}

	var foreignKey []et.Json = []et.Json{}
	for _, v := range m.ForeignKey {
		foreignKey = append(foreignKey, v.Describe())
	}

	var index []et.Json = []et.Json{}
	for _, v := range m.Index {
		index = append(index, v.Describe())
	}

	var unique []et.Json = []et.Json{}
	for _, v := range m.Unique {
		unique = append(unique, v.Describe())
	}

	var relationTo []et.Json = []et.Json{}
	for _, v := range m.RelationTo {
		relationTo = append(relationTo, v.Describe())
	}

	var details []string = []string{}
	for _, v := range m.Details {
		details = append(details, v.Name)
	}

	var hiddens []string = []string{}
	for _, v := range m.Hidden {
		hiddens = append(hiddens, v.Name)
	}

	var requireds []string = []string{}
	for _, v := range m.Required {
		requireds = append(requireds, v.Name)
	}

	var source string = ""
	if m.ColumnSource != nil {
		source = m.ColumnSource.Name
	}

	result := et.Json{
		"schema":            m.Schema.Name,
		"name":              m.Name,
		"tag":               m.Tag,
		"table":             m.Table,
		"description":       m.Description,
		"columns":           columns,
		"primaryKeys":       primaryKeys,
		"foreignKey":        foreignKey,
		"index":             index,
		"unique":            unique,
		"relationTo":        relationTo,
		"details":           details,
		"hidden":            hiddens,
		"requireds":         requireds,
		"source":            source,
		"useStatus":         m.ColumnStatus != nil,
		"useDataSource":     m.ColumnSource != nil,
		"useCreatedTime":    m.ColumnCreatedTime != nil,
		"useCreatedBy":      m.ColumnCreatedBy != nil,
		"useLastEditedTime": m.ColumnLastEditedTime != nil,
		"useLastEditedBy":   m.ColumnLastEditedBy != nil,
		"useProject":        m.ColumnProject != nil,
		"useSerie":          m.ColumnSerie != nil,
		"integrity":         m.Integrity,
		"version":           m.Version,
	}

	return result
}

/**
* Kind
* @return string
**/
func (m *Model) Kind() string {
	return "model"
}

/**
* Init a model
* @param db *DB
* @return error
**/
func (m *Model) Init(db *DB) error {
	m.DB = db
	return db.InitModel(m)
}

/**
* Column add a column to the model
* @param name string
* @return *Column
**/
func (m *Model) Column(name string) *Column {
	idx := IndexColumn(m, name)
	if idx != -1 {
		return m.Columns[idx]
	}

	return nil
}

/**
* Col short to find a column in the model
* @param name string
* @return *Column
**/
func (m *Model) Col(name string) *Column {
	return m.Column(name)
}

/**
* C short to find a column in the model
* @param name string
* @return *Column
**/
func (m *Model) C(name string) *Column {
	return m.Column(name)
}

/**
* AddColumn add a column to the model
* @param col *Column
**/
func (m *Model) AddColumn(col *Column) {
	m.Columns = append(m.Columns, col)
}

/**
* AddUnique add unique column by name to the model
* @param name string
* @param asc bool
* @return *Column
**/
func (m *Model) AddUnique(name string, asc bool) *Column {
	col := Col(m, name)
	if col != nil {
		col.Unique = true
		m.Unique = append(m.Unique, &Index{
			Column: col,
			Asc:    asc,
		})

		return col
	}

	return nil
}

/**
* AddIndex add index column by name to the model
* @param name string
* @param asc bool
* @return *Column
**/
func (m *Model) AddIndex(name string, asc bool) *Column {
	col := Col(m, name)
	if col != nil {
		col.Indexed = true
		m.Index = append(m.Index, &Index{
			Column: col,
			Asc:    asc,
		})

		return col
	}

	return nil
}

/**
* AddPrimaryKey add primary key column by name to the model
* @param name string
* @return *Column
**/
func (m *Model) AddPrimaryKey(name string) *Column {
	col := Col(m, name)
	if col != nil {
		col.Unique = true
		col.PrimaryKey = true
		m.PrimaryKeys = append(m.PrimaryKeys, col)

		return col
	}

	return nil
}

/**
* AddForeignKey add foreign key column by name to the model
* @param foreignKey []string
* @param parentModel *Model
* @param parentKey []string
* @return *Constraint
**/
func (m *Model) AddForeignKey(foreignKey []string, parentModel *Model, parentKey []string) *Constraint {
	for _, v := range m.ForeignKey {
		if v.Fkey() == strings.Join(foreignKey, "_") {
			return v
		}
	}

	for _, n := range foreignKey {
		colA := m.AddIndex(n, true)
		if colA == nil {
			return nil
		}

		parentModel.AddRelationTo(colA)
		colA.ForeignKey = true
	}

	for _, n := range parentKey {
		colB := parentModel.AddIndex(n, true)
		if colB == nil {
			return nil
		}
	}

	result := &Constraint{
		ForeignKey: foreignKey,
		Parent:     parentModel,
		ParentKey:  parentKey,
	}

	m.ForeignKey = append(m.ForeignKey, result)

	return result
}

/**
* AddRelationTo add relation to column to the model
* @param col *Column
**/
func (m *Model) AddRelationTo(col *Column) {
	for _, v := range m.RelationTo {
		if v == col {
			return
		}
	}

	m.RelationTo = append(m.RelationTo, col)
}

/**
* GetMigrateId get migrate id
* @param old_id string
* @return string
* @return error
**/
func (m *Model) GetMigrateId(old_id string) (string, error) {
	return m.DB.GetMigrateId(old_id, m.Table)
}

/**
* UpSertMigrateId upsert migrate id
* @param old_id string
* @param _id string
* @return error
**/
func (m *Model) UpSertMigrateId(old_id, _id string) error {
	return m.DB.UpSertMigrateId(old_id, _id, m.Table)
}
