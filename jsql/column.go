package jsql

import "github.com/cgalvisleon/et/et"

// Default column name

const (
	RESULT     string = "result"
	SOURCE     string = "_source"
	ID         string = "id"
	IDX        string = "_idx"
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
* str: Returns the string representation of the TypeColumn.
* @return string
**/
func (s TypeColumn) str() string {
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

const (
	OF_SYSTEM  string = "of_system"
	ACTIVE     string = "active"
	ARCHIVED   string = "archived"
	CANCELED   string = "canceled"
	FOR_DELETE string = "for_delete"
	// Workflow status
	IN_PROCESS string = "in_process"
	PENDING    string = "pending"
	APPROVED   string = "approved"
	REJECTED   string = "rejected"
	FAILED     string = "failed"
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

var IsEditableStatus = []interface{}{ACTIVE, PENDING}

func statusList() []interface{} {
	return []interface{}{ACTIVE, ARCHIVED, CANCELED, OF_SYSTEM, FOR_DELETE, PENDING, APPROVED, REJECTED}
}

/**
* Column: Describes a single field in a Model, including its storage type, data type, and default.
**/
type Column struct {
	Name       string      `json:"name"`
	TypeColumn TypeColumn  `json:"type_column"`
	TypeData   et.TypeData `json:"type_data"`
	Default    any         `json:"default"`
	model      *Model      `json:"-"`
}

/**
* loadColumn: Loads a column from a JSON object.
* @param params et.Json
* @return *Column
**/
func loadColumn(params et.Json) *Column {
	name := params.String("name")
	typeColumn := params.String("type_column")
	typeData := params.String("type_data")
	defaultValue := params.Any("default")

	return &Column{
		Name:       name,
		TypeColumn: TypeColumn(typeColumn),
		TypeData:   et.TypeData(typeData),
		Default:    defaultValue,
	}
}

/**
* toJson: Returns the column metadata as an et.Json map.
* @return et.Json
**/
func (s *Column) toJson() et.Json {
	return et.Json{
		"name":        s.Name,
		"type_column": s.TypeColumn,
		"type_data":   s.TypeData,
		"default":     s.Default,
	}
}
