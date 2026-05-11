package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

type Define struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Version int      `json:"version"`
	Columns []Column `json:"columns"`
}

/**
* DefineModel: Creates a new model from a declarative Define struct.
* @param define Define
* @return *Model, error
**/
func (s *DB) DefineModel(define Define) (*Model, error) {
	if !utility.ValidStr(define.Schema, 0, []string{}) {
		return nil, errors.New(msg.MSG_SCHEMA_REQUIRED)
	}
	if !utility.ValidStr(define.Name, 0, []string{}) {
		return nil, errors.New(msg.MSG_NAME_REQUIRED)
	}
	if define.Version <= 0 {
		define.Version = 1
	}

	result, err := s.NewModel(define.Schema, define.Name, define.Version)
	if err != nil {
		return nil, err
	}

	for _, column := range define.Columns {
		_, err := result.defineColumn(column.Name, column.TypeColumn, column.TypeData, column.Default, column.Definition)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* defineColumn: Appends a new column definition to the model.
* @param name string
* @param tpColumn TypeColumn
* @param tpData TypeData
* @param def interface{}
* @param definition []byte
* @return *Column, error
**/
func (s *Model) defineColumn(name string, tpColumn TypeColumn, tpData TypeData, def interface{}, definition []byte) (*Column, error) {
	result := &Column{
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    def,
		Definition: definition,
		model:      s,
	}
	s.Columns = append(s.Columns, result)
	return result, nil
}
