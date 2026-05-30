package jsql

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type DefIndex struct {
	Name   string `json:"name"`
	Sorted bool   `json:"sorted"`
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
	Rows int               `json:"rows"`
}

type DefRollup struct {
	Name   string            `json:"name"`
	To     DefTo             `json:"to"`
	Keys   map[string]string `json:"keys"`
	Select []string          `json:"select"`
}

type Def struct {
	Schema      string               `json:"schema"`
	Name        string               `json:"name"`
	Version     int                  `json:"version"`
	IdxField    string               `json:"idx_field"`
	IdtField    string               `json:"idt_field"`
	PrimaryKeys []DefIndex           `json:"primary_keys"`
	ForeignKeys []DefForeignKeys     `json:"foreign_keys"`
	Indexes     []DefIndex           `json:"indexes"`
	Unique      []DefIndex           `json:"unique"`
	Required    []DefIndex           `json:"required"`
	Columns     []Column             `json:"columns"`
	SourceField string               `json:"source_field"`
	Hiddens     []string             `json:"hiddens"`
	Details     map[string]DefDetail `json:"details"`
	Rollups     map[string]DefRollup `json:"rollups"`
	IsCore      bool                 `json:"is_core"`
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

	if s.IdxField != "" {
		pos := s.indexColumn(s.IdxField)
		if pos != -1 {
			s.Columns = append(s.Columns[:pos], append([]*Column{result}, s.Columns[pos:]...)...)
		} else {
			s.Columns = append(s.Columns, result)
		}
	} else {
		s.Columns = append(s.Columns, result)
	}
	return result
}

/**
* DefineSource: Defines the source column for the model.
* @return *Column
**/
func (s *Model) DefineSource() *Column {
	s.SourceField = SOURCE
	return s.defineColumn(SOURCE, COLUMN, JSON, et.Json{}, []byte{})
}

/**
* DefineIdxField: Defines the idx field column for the model.
* @return *Index
**/
func (s *Model) DefineIdxField() *Index {
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
* DefineIdTField: Defines the idt field column for the model.
* @return *Index
**/
func (s *Model) DefineIdTField() *Index {
	s.IdtField = IDT
	result := s.DefineIndex(IDT, INT, 0)
	s.Hiddens = append(s.Hiddens, IDT)
	s.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		now := timezone.Now()
		new[s.IdtField] = now.UnixMilli()
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
* @param name string, keys map[string]string, rows int
* @return (*Model, error)
**/
func (s *Model) DefineDetail(name string, keys map[string]string, rows int) (*Model, error) {
	result, ok := s.Details[name]
	if ok {
		return result.To.Model, nil
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf(msg.MSG_KEYS_REQUIRED)
	}

	detailName := fmt.Sprintf("%s_%s", s.Name, name)
	to, err := s.db.NewModel(s.Schema, detailName, 1)
	if err != nil {
		return nil, err
	}
	for k, fk := range keys {
		s.defineColumn(k, COLUMN, KEY, "", []byte{})
		to.DefinePrimaryKey(fk, KEY, "")
		to.DefineHidden(fk)
	}
	s.defineColumn(name, DETAIL, ANY, nil, []byte{})
	detail := newDetail(to, keys, []string{}, true, true)
	detail.Rows = rows
	s.Details[name] = detail
	return to, nil
}

/**
* DefineRollup: Defines a new rollup for the model.
* @param name string, to *Model,
* @param keys map[string]string is primary key and foreign key,
* @param selects []string
* @return (*Detail, error)
**/
func (s *Model) DefineRollup(name string, to *Model, keys map[string]string, selects []string) (*Detail, error) {
	result, ok := s.Details[name]
	if ok {
		return result, nil
	}

	if to == nil {
		return nil, fmt.Errorf(msg.MSG_TO_MODEL_REQUIRED)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf(msg.MSG_KEYS_REQUIRED)
	}

	if len(selects) == 0 {
		return nil, fmt.Errorf(msg.MSG_SELECTS_REQUIRED)
	}

	s.defineColumn(name, ROLLUP, ANY, nil, []byte{})
	detail := newDetail(to, keys, selects, false, false)
	detail.Rows = 1
	s.Details[name] = detail
	return detail, nil
}

/**
* DefineCalc: Defines a new calculation for the model.
* @param name string, calc CalcFunction
* @return *Model
**/
func (s *Model) DefineCalc(name string, calc CalcFunction) *Model {
	s.defineColumn(name, CALC, ANY, nil, []byte{})
	s.Calcs[name] = calc
	return s
}

/**
* DefineModel: Defines the standard columns for the model.
* @return *Model
**/
func (s *Model) DefineModel() *Model {
	s.DefineColumn(CREATED_AT, DATETIME, nil)
	s.DefineColumn(UPDATED_AT, DATETIME, nil)
	s.DefineIndex(STATUS, TEXT, ACTIVE)
	s.DefinePrimaryKey(ID, KEY, "")
	s.DefineSource()
	s.DefineIdxField()
	return s
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
	result.DefineModel()
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
	result.DefineModel()
	result.DefineIndex(TENANT_ID, KEY, "")
	result.DefineSource()
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
	result.DefineModel()
	result.DefineIndex(PROJECT_ID, KEY, "")
	result.DefineSource()
	return result, nil
}
