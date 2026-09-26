package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* defaultTrigger: Registers the triggers that enforce the unique fields of the model (DefineUnique)
* before each insert and update. The check runs inside the command's transaction, so it also sees
* the rows written earlier in the same transaction (e.g. a bulk insert) and does not open another
* connection (SQLite has a single writer connection). A duplicate returns ErrRecordAlreadyExists
* with the field and value. The unique index of the database remains as the final guarantee.
* @return *Model
**/
func (s *Model) defaultTrigger() *Model {
	s.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		for _, unique := range s.Unique {
			value, ok := new[unique.Name]
			if !ok || isEmptyUnique(value) {
				continue
			}
			exists, err := s.Where(Eq(unique.Name, value)).ExistsTx(tx)
			if err != nil {
				return err
			}
			if exists {
				return uniqueError(unique.Name, value)
			}
		}
		return nil
	})

	s.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		for _, unique := range s.Unique {
			value, ok := new[unique.Name]
			if !ok || isEmptyUnique(value) || !old.IsDeferent(unique.Name, value) {
				continue
			}
			// Other rows with the value; the row being updated is told apart by its primary key.
			rows, err := s.Where(Eq(unique.Name, value)).LimitTx(tx, 1, 2)
			if err != nil {
				return err
			}
			for _, row := range rows.Result {
				if !s.samePrimaryKey(row, old) {
					return uniqueError(unique.Name, value)
				}
			}
		}
		return nil
	})

	return s
}

/**
* samePrimaryKey: Reports whether two records have the same primary key values.
* @param a, b et.Json
* @return bool
**/
func (s *Model) samePrimaryKey(a, b et.Json) bool {
	if len(s.PrimaryKeys) == 0 {
		return false
	}
	for _, pk := range s.PrimaryKeys {
		if fmt.Sprint(a[pk.Name]) != fmt.Sprint(b[pk.Name]) {
			return false
		}
	}
	return true
}

/**
* isEmptyUnique: Reports whether a unique value is empty (nil or ""), which is not checked.
* @param value any
* @return bool
**/
func isEmptyUnique(value any) bool {
	return value == nil || value == ""
}

/**
* uniqueError: Returns ErrRecordAlreadyExists with the duplicated field and value.
* @param field string, value any
* @return error
**/
func uniqueError(field string, value any) error {
	return fmt.Errorf("%w: %s = %v", ErrRecordAlreadyExists, field, value)
}
