package jsql

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
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
	Name        string            `json:"name"`
	Keys        map[string]string `json:"keys"`
	Rows        int               `json:"rows"`
	Select      []string          `json:"select"`
	Columns     []Column          `json:"columns"`
	PrimaryKeys []DefIndex        `json:"primary_keys"`
	Indexes     []DefIndex        `json:"indexes"`
	Rollups     []DefRollup       `json:"rollups"`
	IdxField    string            `json:"idx_field"`
	IdtField    string            `json:"idt_field"`
}

type DefMaster struct {
	Name   string            `json:"name"`
	To     DefTo             `json:"to"`
	Keys   map[string]string `json:"keys"`
	ToKeys map[string]string `json:"to_keys"`
	Select []string          `json:"select"`
	Rows   int               `json:"rows"`
}

type DefRollup struct {
	Name      string            `json:"name"`
	To        DefTo             `json:"to"`
	Keys      map[string]string `json:"keys"`
	Select    []string          `json:"select"`
	Operation RollupOperation   `json:"operation"`
}

type Define struct {
	Schema      string           `json:"schema"`
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Version     int              `json:"version"`
	SourceField string           `json:"source_field"`
	IdxField    string           `json:"idx_field"`
	PrimaryKeys []DefIndex       `json:"primary_keys"`
	ForeignKeys []DefForeignKeys `json:"foreign_keys"`
	Indexes     []DefIndex       `json:"indexes"`
	Unique      []DefIndex       `json:"unique"`
	Required    []DefIndex       `json:"required"`
	Columns     []Column         `json:"columns"`
	Hiddens     []string         `json:"hiddens"`
	Details     []DefDetail      `json:"details"`
	Masters     []DefMaster      `json:"master"`
	Rollups     []DefRollup      `json:"rollups"`
	UserId      string           `json:"user_id"`
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
* @param name string, tpColumn TypeColumn, tpData et.TypeData, default any
* @return *Column
**/
func (s *Model) defineColumn(name string, tpColumn TypeColumn, tpData et.TypeData, deFault any) *Column {
	idx := s.indexColumn(name)
	if idx != -1 {
		return s.Columns[idx]
	}

	result := &Column{
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    deFault,
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
* defineSource: Defines the source column for the model.
* @return *Column
**/
func (s *Model) defineSource() *Column {
	s.SourceField = SOURCE
	return s.defineColumn(SOURCE, COLUMN, et.JSON, et.Json{})
}

/**
* defineIdxField: Defines the idx field column for the model.
* @return *Index
**/
func (s *Model) defineIdxField() *Index {
	s.IdxField = IDX
	result := s.DefineIndex(IDX, et.KEY, "")
	s.Hiddens = append(s.Hiddens, IDX)
	s.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		new[s.IdxField] = s.getIdx()
		return nil
	})

	return result
}

/**
* defineIndex: Defines a new index column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) defineIndex(name string, tp et.TypeData, deFault any) *Index {
	s.defineColumn(name, COLUMN, tp, deFault)
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
* definePrimaryKey: Defines a new primary key column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) definePrimaryKey(name string, tp et.TypeData, deFault any) *Index {
	s.defineColumn(name, COLUMN, tp, deFault)
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
* defineForeignKeys: Defines a new foreign key column for the model.
* @param to *Model, keys map[string]string, onDeleteCascade bool, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) defineForeignKeys(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) *Detail {
	idx := slices.IndexFunc(s.ForeignKeys, func(idx *Detail) bool { return idx.To.Name == to.Name })
	if idx != -1 {
		return s.ForeignKeys[idx]
	}
	detail := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.ForeignKeys = append(s.ForeignKeys, detail)
	return detail
}

/**
* defineUnique: Defines a new unique index for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) defineUnique(name string, tp et.TypeData, deFault any) *Index {
	s.defineColumn(name, COLUMN, tp, deFault)
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
* defineRequired: Defines a new required column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) defineRequired(name string, tp et.TypeData, deFault any) *Index {
	s.defineColumn(name, COLUMN, tp, deFault)
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
* defineHidden: Defines a new hidden column for the model.
* @param name ...string
**/
func (s *Model) defineHidden(name ...string) {
	s.Hiddens = append(s.Hiddens, name...)
}

/**
* defineRealColumn: Defines a new column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Column
**/
func (s *Model) defineRealColumn(name string, tp et.TypeData, deFault any) *Column {
	return s.defineColumn(name, COLUMN, tp, deFault)
}

/**
* defineAttrib: Defines a new attribute for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Column
**/
func (s *Model) defineAttrib(name string, tp et.TypeData, deFault any) *Column {
	return s.defineColumn(name, ATTRIB, tp, deFault)
}

/**
* defineRollup: Defines a new rollup for the model.
* Selects are required except for RollupCount (counts rows) and RollupObject (whole row).
* An empty operation defaults to RollupObject.
* @param name string, to *Model, keys map[string]string, selects []string, operation RollupOperation
* @return (*Rollups, error)
**/
func (s *Model) defineRollup(name string, to *Model, keys map[string]string, selects []string, operation RollupOperation) (*Rollups, error) {
	result, ok := s.Rollups[name]
	if ok {
		return result, nil
	}

	if to == nil {
		return nil, errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if len(keys) == 0 {
		return nil, errors.New(MSG_KEYS_REQUIRED)
	}

	if operation == "" {
		operation = RollupObject
	}

	if !operation.IsValid() {
		return nil, fmt.Errorf(MSG_INVALID_ROLLUP_OPERATION, operation)
	}

	if len(selects) == 0 && operation != RollupCount && operation != RollupObject {
		return nil, errors.New(MSG_SELECTS_REQUIRED)
	}

	s.defineColumn(name, ROLLUP, et.ANY, nil)
	result = newRollup(to, keys, selects, operation)
	s.Rollups[name] = result
	return result, nil
}

/**
* defineDetail: Defines a new detail for the model.
* @param name string, keys map[string]string, rows int
* @return (*Model, error)
**/
func (s *Model) defineDetail(name string, keys map[string]string, rows int, selects ...string) (*Model, error) {
	result, ok := s.Details[name]
	if ok {
		return result.To.Model, nil
	}

	if len(keys) == 0 {
		return nil, errors.New(MSG_KEYS_REQUIRED)
	}

	detailName := fmt.Sprintf("%s_%s", s.Name, name)
	to := s.db.NewModel(s.Schema, detailName, 1, s.ID)
	for k, fk := range keys {
		s.defineColumn(k, COLUMN, et.KEY, "")
		to.defineColumn(fk, COLUMN, et.KEY, "")
		to.DefineForeignKeys(s, map[string]string{fk: k}, true, false)
		to.DefineHidden(fk)
	}
	s.defineColumn(name, DETAIL, et.ANY, nil)
	detail := newDetail(to, keys, selects, true, true)
	detail.Rows = rows
	s.Details[name] = detail
	return to, nil
}

/**
* defineMaster: Defines a new master for the model.
* @param name string, to *Model, keys, toKeys map[string]string, selects []string
* @return (*Master, error)
**/
func (s *Model) defineMaster(name string, to *Model, keys, toKeys map[string]string, selects []string, rows ...int) (*Model, error) {
	result, ok := s.Masters[name]
	if ok {
		return result.To.Model, nil
	}

	if len(keys) == 0 {
		return nil, errors.New(MSG_KEYS_REQUIRED)
	}

	detailName := fmt.Sprintf("%s_%s", s.Name, to.Name)
	bridge := s.db.NewModel(s.Schema, detailName, 1, s.ID)
	bridge.DefineIdxField()
	for k, fk := range keys {
		bridge.DefinePrimaryKey(fk, et.KEY, "")
		bridge.DefineForeignKeys(s, map[string]string{fk: k}, true, false)
	}
	for k, fk := range toKeys {
		bridge.DefinePrimaryKey(fk, et.KEY, "")
		bridge.DefineForeignKeys(to, map[string]string{fk: k}, true, false)
	}
	s.defineColumn(name, MASTER, et.ANY, nil)
	master := newMaster(s, to, bridge, keys, toKeys, selects)
	if len(rows) > 0 {
		master.Rows = rows[0]
	}
	s.Masters[name] = master
	to.Masters[s.Name] = master
	return bridge, nil
}

/**
* defineCalcFunc: Defines a new calculation for the model.
* @param name string, calc CalcFunction
* @return *Model
**/
func (s *Model) defineCalcFunc(name string, calc CalcFunction) *Model {
	s.defineColumn(name, CALCFUNC, et.ANY, nil)
	s.calcs[name] = calc
	return s
}

/**
* defineCalc: Defines a new calculation for the model using a bytecode definition.
* @param name string, code string
* @return *Model
**/
func (s *Model) defineCalc(name, script string) *Model {
	s.defineColumn(name, CALC, et.ANY, nil)
	s.calcScripts[name] = script
	return s
}

/**
* defineBeforeInsert: (stores the JS code; name is used when code is empty) Defines a new before insert hook for the model.
* @param name string
* @return *Model
**/
func (s *Model) defineBeforeInsert(name string) *Model {
	idx := slices.IndexFunc(s.BeforeInserts, func(r string) bool { return r == name })
	if idx != -1 {
		s.BeforeInserts[idx] = name
	} else {
		s.BeforeInserts = append(s.BeforeInserts, name)
	}
	return s
}

/**
* defineBeforeUpdate: (stores the JS code; name is used when code is empty) Defines a new before update hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) defineBeforeUpdate(name, code string) *Model {
	if code == "" {
		code = name
	}
	if !slices.Contains(s.BeforeUpdates, code) {
		s.BeforeUpdates = append(s.BeforeUpdates, code)
	}
	return s
}

/**
* defineBeforeDelete: (stores the JS code; name is used when code is empty) Defines a new before delete hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) defineBeforeDelete(name, code string) *Model {
	if code == "" {
		code = name
	}
	if !slices.Contains(s.BeforeDeletes, code) {
		s.BeforeDeletes = append(s.BeforeDeletes, code)
	}
	return s
}

/**
* defineAfterInsert: (stores the JS code; name is used when code is empty) Defines a new after insert hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) defineAfterInsert(name, code string) *Model {
	if code == "" {
		code = name
	}
	if !slices.Contains(s.AfterInserts, code) {
		s.AfterInserts = append(s.AfterInserts, code)
	}
	return s
}

/**
* defineAfterUpdate: (stores the JS code; name is used when code is empty) Defines a new after update hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) defineAfterUpdate(name, code string) *Model {
	if code == "" {
		code = name
	}
	if !slices.Contains(s.AfterUpdates, code) {
		s.AfterUpdates = append(s.AfterUpdates, code)
	}
	return s
}

/**
* defineAfterDelete: (stores the JS code; name is used when code is empty) Defines a new after delete hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) defineAfterDelete(name, code string) *Model {
	if code == "" {
		code = name
	}
	if !slices.Contains(s.AfterDeletes, code) {
		s.AfterDeletes = append(s.AfterDeletes, code)
	}
	return s
}

/**
* defineModel: Defines the standard columns for the model.
* @return *Model
**/
func (s *Model) defineModel() *Model {
	s.DefineColumn(CREATED_AT, et.DATETIME, nil)
	s.DefineColumn(UPDATED_AT, et.DATETIME, nil)
	s.DefineIndex(STATUS, et.TEXT, ACTIVE)
	s.DefinePrimaryKey(ID, et.KEY, "")
	s.DefineSource()
	s.DefineIdxField()
	return s
}

/**
* defineModel: Defines a new model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) defineModel(schema, name string, version int, userId string) (*Model, error) {
	result := s.NewModel(schema, name, version, userId)
	result.DefineModel()
	return result, nil
}

/**
* defineTenantModel: Defines a new tenant model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) defineTenantModel(schema, name string, version int, userId string) (*Model, error) {
	result := s.NewModel(schema, name, version, userId)
	result.DefineModel()
	result.DefineIndex(TENANT_ID, et.KEY, "")
	result.DefineSource()
	return result, nil
}

/**
* defineProjectModel: Defines a new project model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) defineProjectModel(schema, name string, version int, userId string) (*Model, error) {
	result := s.NewModel(schema, name, version, userId)
	result.DefineModel()
	result.DefineIndex(PROJECT_ID, et.KEY, "")
	result.DefineSource()
	return result, nil
}
