package oracle

import (
	"database/sql"

	"github.com/cgalvisleon/et/jsql"
)

/**
* ExistModel: Returns true when the model's table exists. Not implemented yet.
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Oracle) ExistModel(db *sql.DB, model *jsql.Model) (bool, error) {
	return false, errNotImplemented
}

/**
* Load: Generates the DDL SQL for the given model. Not implemented yet.
* @param model *jsql.Model
* @return string, error
**/
func (s *Oracle) Load(model *jsql.Model) (string, error) {
	return "", errNotImplemented
}
