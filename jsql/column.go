package jsql

const (
	RESULT     string = "result"
	SOURCE     string = "_source"
	IDX        string = "_idx"
	ID         string = "id"
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
	COLUMN TypeColumn = "column"
	ATTRIB TypeColumn = "atrib"
	DETAIL TypeColumn = "detail"
	ROLLUP TypeColumn = "rollup"
	CALC   TypeColumn = "calc"
	AGG    TypeColumn = "agg"
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
