package jsql

import (
	"database/sql"
	"regexp"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

// numberScale reads the precision and scale of a NUMBER/NUMERIC/DECIMAL type: "number(19)", "numeric(10,2)".
var numberScale = regexp.MustCompile(`^(?:number|numeric|decimal)\s*\(\s*\d+\s*(?:,\s*(-?\d+)\s*)?\)`)

/**
* typeDataOf: Maps a SQL type read from a catalog to the et.TypeData closest to it.
* @param sqlType string
* @return et.TypeData
**/
func typeDataOf(sqlType string) et.TypeData {
	t := strings.ToLower(strings.TrimSpace(sqlType))
	has := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(t, part) {
				return true
			}
		}
		return false
	}
	switch {
	case t == "":
		return et.ANY
	case strings.HasPrefix(t, "interval"):
		return et.TEXT
	case strings.HasSuffix(t, "[]") || t == "array":
		return et.ARRAY
	case has("json"):
		return et.JSON
	case t == "bit" || t == "tinyint(1)" || has("bool"):
		return et.BOOL
	case has("blob", "bytea", "binary", "image") || t == "raw" || strings.HasPrefix(t, "raw("):
		return et.BYTE
	case has("int", "serial"):
		return et.INT
	case numberScale.MatchString(t):
		if scale := numberScale.FindStringSubmatch(t)[1]; scale == "" || scale == "0" {
			return et.INT
		}
		return et.FLOAT
	case has("number", "numeric", "decimal", "real", "double", "float", "money"):
		return et.FLOAT
	case has("date", "time"):
		return et.DATETIME
	case has("clob", "text", "(max)"):
		return et.MEMO
	default:
		return et.TEXT
	}
}

/**
* reflectModel: Builds the model of an existing table that this process did not define, from the database
* catalog (Describer), and registers it in its schema so the next queries and commands find it. Columns named
* _source and _idx become the SourceField and the IdxField, so the attributes of a jsql table can be used. The
* model is marked initialized: its table is never created nor changed.
* @param schema, name string
* @return *Model (nil when the table does not exist or the driver cannot describe), error
**/
func (s *DB) reflectModel(schema, name string) (*Model, error) {
	describer, ok := s.driver.(Describer)
	if !ok || s.db == nil {
		return nil, nil
	}

	schema = utility.Normalize(schema)
	sch, ok := s.lookupSchema(schema)
	if !ok {
		sch = &Schema{Name: schema, Models: map[string]*Model{}, database: s.Name, db: s, isDebug: s.isDebug, mu: &sync.RWMutex{}}
	}
	model := sch.buildModel("", name, 1)
	columns, err := describer.Describe(s.db, model, s.timeout)
	if err != nil || len(columns) == 0 {
		return nil, err
	}

	for _, col := range columns {
		switch col.Name {
		case SOURCE:
			model.defineSource()
		case IDX:
			model.defineIdxField()
		default:
			model.defineColumn(col.Name, COLUMN, typeDataOf(col.Type), nil)
		}
	}
	model.IsInit = true

	sch = s.getSchema(schema)
	sch.mu.Lock()
	defer sch.mu.Unlock()
	if existing, ok := sch.Models[model.Name]; ok {
		return existing, nil
	}
	model.schema = sch
	sch.Models[model.Name] = model
	return model, nil
}

/**
* describeRows: Reads the rows of a catalog query (columns name and type) as described columns.
* @param rows *sql.Rows, err error
* @return []DescribedColumn, error
**/
func describeRows(rows *sql.Rows, err error) ([]DescribedColumn, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []DescribedColumn{}
	for rows.Next() {
		var name, tp sql.NullString
		if err := rows.Scan(&name, &tp); err != nil {
			return nil, err
		}
		if name.String != "" {
			result = append(result, DescribedColumn{Name: name.String, Type: tp.String})
		}
	}
	return result, rows.Err()
}
