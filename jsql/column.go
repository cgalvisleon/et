package jsql

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

var Status = map[string]bool{
	ARCHIVED:   true,
	CANCELED:   true,
	OF_SYSTEM:  true,
	FOR_DELETE: true,
	PENDING:    true,
	APPROVED:   true,
	REJECTED:   true,
}

/**
* SetStatus: Adds a new status to the Status map.
* @param status string
**/
func SetStatus(status string) {
	Status[status] = true
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
* up: Internal method to associate the column with a model and return it for chaining.
* @param model *Model
* @return *Column
**/
func (s *Column) up(model *Model) *Column {
	s.model = model
	return s
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
