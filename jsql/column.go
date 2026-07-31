package jsql

import "github.com/cgalvisleon/et/et"

const (
	RESULT     string = "result"
	SOURCE     string = "_source"
	ID         string = "id"
	IDX        string = "_idx"
	IDT        string = "_idt"
	STATUS     string = "status"
	VERSION    string = "version"
	TENANT_ID  string = "tenant_id"
	PROJECT_ID string = "project_id"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

/**
* TypeColumn: Classifies how a column is stored (real column, JSONB attribute, relation, etc.).
**/
type TypeColumn string

/**
* Str: Returns the string representation of the TypeColumn.
* @return string
**/
func (s TypeColumn) Str() string {
	return string(s)
}

const (
	COLUMN   TypeColumn = "column"
	ATTRIB   TypeColumn = "atrib"
	DETAIL   TypeColumn = "detail"
	MASTER   TypeColumn = "master"
	ROLLUP   TypeColumn = "rollup"
	CALCFUNC TypeColumn = "calc_func"
	CALC     TypeColumn = "calc"
	AGG      TypeColumn = "agg"
)

/**
* TypeData: Specifies the logical data type of a column value.
**/
type TypeData string

/**
* Str: Returns the string representation of the TypeData.
* @return string
**/
func (s TypeData) Str() string {
	return string(s)
}

const (
	ANY       TypeData = "any"
	BYTES     TypeData = "bytes"
	INT       TypeData = "int"
	FLOAT     TypeData = "float"
	KEY       TypeData = "key"
	TEXT      TypeData = "text"
	MEMO      TypeData = "memo"
	JSON      TypeData = "json"
	DATETIME  TypeData = "datetime"
	BOOLEAN   TypeData = "boolean"
	GEOMETRY  TypeData = "geometry"
	EMBEDDING TypeData = "embedding"
)

const (
	ACTIVE     string = "active"
	ARCHIVED   string = "archived"
	CANCELED   string = "canceled"
	OF_SYSTEM  string = "of_system"
	FOR_DELETE string = "for_delete"
	PENDING    string = "pending"
	APPROVED   string = "approved"
	REJECTED   string = "rejected"
)

var Status = map[string]bool{
	ACTIVE:     true,
	ARCHIVED:   true,
	CANCELED:   true,
	OF_SYSTEM:  true,
	FOR_DELETE: true,
	PENDING:    true,
	APPROVED:   true,
	REJECTED:   true,
}

func StatusList() []interface{} {
	return []interface{}{ACTIVE, ARCHIVED, CANCELED, OF_SYSTEM, FOR_DELETE, PENDING, APPROVED, REJECTED}
}

/**
* Column: Describes a single field in a Model, including its storage type, data type, and default.
**/
type Column struct {
	Name       string     `json:"name"`
	TypeColumn TypeColumn `json:"type_column"`
	TypeData   TypeData   `json:"type_data"`
	Default    any        `json:"default"`
	Definition []byte     `json:"definition"`
	model      *Model     `json:"-"`
}

/**
* Ref: Returns the reference of the column.
* @return et.Json
**/
func (s *Column) Ref() et.Json {
	return et.Json{
		"name":        s.Name,
		"type_column": s.TypeColumn,
		"type_data":   s.TypeData,
		"default":     s.Default,
		"definition":  s.Definition,
	}
}

/**
* SetModel: Associates the column with the given model and returns the column for chaining.
* @param model *Model
* @return *Column
**/
func (s *Column) SetModel(model *Model) *Column {
	s.model = model
	return s
}

/**
* SetDefinition: Sets the raw definition bytes on the column and returns it for chaining.
* @param definition []byte
* @return *Column
**/
func (s *Column) SetDefinition(definition []byte) *Column {
	s.Definition = definition
	return s
}
