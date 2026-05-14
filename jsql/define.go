package jsql

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

type DefIndex struct {
	Name     string   `json:"name"`
	TypeData TypeData `json:"type_data"`
	Default  any      `json:"default"`
}

type DefTo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type DefForeignKeys struct {
	To              DefTo             `json:"to"`
	Keys            map[string]string `json:"keys"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

type DefDetail struct {
	Name string            `json:"name"`
	Keys map[string]string `json:"keys"`
}

type DefRollup struct {
	Name   string            `json:"name"`
	To     DefTo             `json:"to"`
	Keys   map[string]string `json:"keys"`
	Select []string          `json:"select"`
}

type Define struct {
	Schema      string               `json:"schema"`
	Name        string               `json:"name"`
	Version     int                  `json:"version"`
	Columns     []Column             `json:"columns"`
	SourceField string               `json:"source_field"`
	IdxField    string               `json:"idx_field"`
	PrimaryKeys []DefIndex           `json:"primary_keys"`
	ForeignKeys []DefForeignKeys     `json:"foreign_keys"`
	Indexes     []DefIndex           `json:"indexes"`
	Unique      []DefIndex           `json:"unique"`
	Required    []DefIndex           `json:"required"`
	Hiddens     []string             `json:"hiddens"`
	Details     map[string]DefDetail `json:"details"`
	Rollups     map[string]DefRollup `json:"rollups"`
	IsDebug     bool                 `json:"is_debug"`
	IsTest      bool                 `json:"is_test"`
}

/**
* indexColumn: Returns the index of the column with the given name.
* @param name string
* @return int
**/
func (s *Model) indexColumn(name string) int {
	result := slices.IndexFunc(s.Columns, func(col *Column) bool { return col.Name == name })
	return result
}

/**
* defineColumn: Appends a new column definition to the model.
* @param name string, tpColumn TypeColumn, tpData TypeData, def any, definition []byte
* @return *Column
**/
func (s *Model) defineColumn(name string, tpColumn TypeColumn, tpData TypeData, def any, definition []byte) *Column {
	idx := s.indexColumn(name)
	if idx != -1 {
		return s.Columns[idx]
	}

	result := &Column{
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    def,
		Definition: definition,
		model:      s,
	}
	s.Columns = append(s.Columns, result)
	return result
}

/**
* defineSource: Defines the source column for the model.
* @return *Column
**/
func (s *Model) defineSource() *Column {
	s.SourceField = SOURCE
	return s.defineColumn(SOURCE, COLUMN, JSON, et.Json{}, []byte{})
}

/**
* defineIdxField: Defines the idx field column for the model.
* @return *Index
**/
func (s *Model) defineIdxField() *Index {
	s.IdxField = IDX
	result := s.DefineIndex(IDX, KEY, "")
	s.Hiddens = append(s.Hiddens, IDX)
	s.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		new[s.IdxField] = reg.GetULID("")
		return nil
	})

	return result
}

/**
* DefineIndex: Defines a new index column for the model.
* @param name string, tp TypeData, def any
* @return *Index
**/
func (s *Model) DefineIndex(name string, tp TypeData, def any) *Index {
	s.defineColumn(name, COLUMN, tp, def, []byte{})
	idx := slices.IndexFunc(s.Indexes, func(idx *Index) bool { return idx.Name == name })
	if idx != -1 {
		return s.Indexes[idx]
	}
	index := &Index{
		Name:   name,
		Sorted: true,
	}
	s.Indexes = append(s.Indexes, index)
	return index
}

/**
* DefinePrimaryKey: Defines a new primary key column for the model.
* @param name string, tp TypeData, def any
* @return *Index
**/
func (s *Model) DefinePrimaryKey(name string, tp TypeData, def any) *Index {
	s.defineColumn(name, COLUMN, tp, def, []byte{})
	idx := slices.IndexFunc(s.PrimaryKeys, func(idx *Index) bool { return idx.Name == name })
	if idx != -1 {
		return s.PrimaryKeys[idx]
	}
	index := &Index{
		Name:   name,
		Sorted: true,
	}
	s.PrimaryKeys = append(s.PrimaryKeys, index)
	return index
}

/**
* DefineForeignKeys: Defines a new foreign key column for the model.
* @param to *Model, keys map[string]string, onDeleteCascade bool, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) DefineForeignKeys(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) *Detail {
	idx := slices.IndexFunc(s.ForeignKeys, func(idx *Detail) bool { return idx.To.Name == to.Name })
	if idx != -1 {
		return s.ForeignKeys[idx]
	}
	detail := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.ForeignKeys = append(s.ForeignKeys, detail)
	return detail
}

/**
* DefineUnique: Defines a new unique index for the model.
* @param name string, tp TypeData, def any
* @return *Index
**/
func (s *Model) DefineUnique(name string, tp TypeData, def any) *Index {
	s.defineColumn(name, COLUMN, tp, def, []byte{})
	idx := slices.IndexFunc(s.Unique, func(idx *Index) bool { return idx.Name == name })
	if idx != -1 {
		return s.Unique[idx]
	}
	index := &Index{
		Name:   name,
		Sorted: true,
	}
	s.Unique = append(s.Unique, index)
	return index
}

/**
* DefineRequired: Defines a new required column for the model.
* @param name string, tp TypeData, def any
* @return *Index
**/
func (s *Model) DefineRequired(name string, tp TypeData, def any) *Index {
	s.defineColumn(name, COLUMN, tp, def, []byte{})
	idx := slices.IndexFunc(s.Required, func(idx *Index) bool { return idx.Name == name })
	if idx != -1 {
		return s.Required[idx]
	}
	index := &Index{
		Name:   name,
		Sorted: true,
	}
	s.Required = append(s.Required, index)
	return index
}

/**
* DefineHidden: Defines a new hidden column for the model.
* @param name ...string
**/
func (s *Model) DefineHidden(name ...string) {
	s.Hiddens = append(s.Hiddens, name...)
}

/**
* DefineColumn: Defines a new column for the model.
* @param name string, tp TypeData, def any
* @return *Column
**/
func (s *Model) DefineColumn(name string, tp TypeData, def any) *Column {
	return s.defineColumn(name, COLUMN, tp, def, []byte{})
}

/**
* DefineAttrib: Defines a new attribute for the model.
* @param name string, tp TypeData, def any
* @return *Column
**/
func (s *Model) DefineAttrib(name string, tp TypeData, def any) *Column {
	return s.defineColumn(name, ATTRIB, tp, def, []byte{})
}

/**
* DefineDetail: Defines a new detail for the model.
* @param name string, keys map[string]string
* @return *Detail
**/
func (s *Model) DefineDetail(name string, keys map[string]string) *Detail {
	result, ok := s.Details[name]
	if ok {
		return result
	}

	detailName := fmt.Sprintf("%s_%s", s.Name, name)
	to, err := s.db.NewModel(s.Schema, detailName, 1)
	if err != nil {
		return nil
	}
	for fk, k := range keys {
		s.defineColumn(fk, COLUMN, KEY, "", []byte{})
		to.defineColumn(k, COLUMN, KEY, "", []byte{})
	}
	s.defineColumn(name, DETAIL, ANY, nil, []byte{})
	detail := newDetail(to, keys, []string{}, true, true)
	s.Details[name] = detail
	return detail
}

/**
* DefineRollup: Defines a new rollup for the model.
* @param name string, to *Model, keys map[string]string, selects []string
* @return *Detail
**/
func (s *Model) DefineRollup(name string, to *Model, keys map[string]string, selects []string) *Detail {
	result, ok := s.Details[name]
	if ok {
		return result
	}

	s.defineColumn(name, ROLLUP, ANY, nil, []byte{})
	detail := newDetail(to, keys, selects, false, false)
	s.Details[name] = detail
	return detail
}

/**
* DefineModel: Defines a new model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineModel(schema, name string, version int) (*Model, error) {
	result, err := s.NewModel(schema, name, version)
	if err != nil {
		return nil, err
	}
	result.DefineColumn(CREATED_AT, DATETIME, nil)
	result.DefineColumn(UPDATED_AT, DATETIME, nil)
	result.DefinePrimaryKey(ID, KEY, "")
	result.defineSource()
	result.defineIdxField()
	return result, nil
}

/**
* DefineTenantModel: Defines a new tenant model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineTenantModel(schema, name string, version int) (*Model, error) {
	result, err := s.NewModel(schema, name, version)
	if err != nil {
		return nil, err
	}
	result.DefineColumn(CREATED_AT, DATETIME, nil)
	result.DefineColumn(UPDATED_AT, DATETIME, nil)
	result.DefineIndex(TENANT_ID, KEY, "")
	result.DefinePrimaryKey(ID, KEY, "")
	result.defineSource()
	result.defineIdxField()
	return result, nil
}

/**
* DefineProjectModel: Defines a new project model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineProjectModel(schema, name string, version int) (*Model, error) {
	result, err := s.NewModel(schema, name, version)
	if err != nil {
		return nil, err
	}
	result.DefineColumn(CREATED_AT, DATETIME, nil)
	result.DefineColumn(UPDATED_AT, DATETIME, nil)
	result.DefineIndex(PROJECT_ID, KEY, "")
	result.DefinePrimaryKey(ID, KEY, "")
	result.defineSource()
	result.defineIdxField()
	return result, nil
}
