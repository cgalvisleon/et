package jsql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type TxStatus string

const (
	TxStatusPending    TxStatus = "pending"
	TxStatusCommitted  TxStatus = "committed"
	TxStatusRolledBack TxStatus = "rolled_back"
)

type Tx struct {
	CreatedAt    time.Time `json:"created_at"`
	LastUpdateAt time.Time `json:"last_update_at"`
	Id           string    `json:"id"`
	Status       string    `json:"status"`
	Tx           *sql.Tx   `json:"-"`
}

/**
* getTx: Returns the given Tx unchanged, or creates a new auto-commit Tx when nil.
* The second return value is true when the caller must commit after the operation.
* @param tx *Tx
* @return *Tx, bool
**/
func getTx(tx *Tx) (*Tx, bool) {
	isCommitted := false
	if tx == nil {
		now := timezone.Now()
		tx = &Tx{
			CreatedAt:    now,
			LastUpdateAt: now,
			Id:           reg.TagULID("tx", ""),
			Status:       string(TxStatusPending),
		}
		isCommitted = true
	}

	return tx, isCommitted
}

/**
* begin: Opens a database transaction on the underlying *sql.DB if not already open.
* @param db *sql.DB
* @return error
**/
func (s *Tx) begin(db *sql.DB) error {
	if s.Tx != nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	s.Tx = tx

	return nil
}

/**
* setStatus: Updates the transaction status and records the change time.
* @param status TxStatus
**/
func (s *Tx) setStatus(status TxStatus) {
	s.Status = string(status)
	s.LastUpdateAt = timezone.Now()
}

/**
* commit: Commits the transaction and marks it as committed.
* @return error
**/
func (s *Tx) commit() error {
	if s.Tx == nil {
		return nil
	}

	err := s.Tx.Commit()
	if err != nil {
		return err
	}

	s.setStatus(TxStatusCommitted)

	return nil
}

/**
* rollback: Rolls back the transaction and marks it as rolled back.
* @return error
**/
func (s *Tx) rollback() error {
	if s.Tx == nil {
		return nil
	}

	err := s.Tx.Rollback()
	if err != nil {
		return err
	}

	s.setStatus(TxStatusRolledBack)

	return nil
}

/**
* Query: Executes a query within the transaction.
* @param db *sql.DB, query string, args ...any
* @return *sql.Rows, error
**/
func (s *Tx) Query(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	err := s.begin(db)
	if err != nil {
		return nil, err
	}

	rows, err := s.Tx.Query(query, args...)
	if err != nil {
		errR := s.rollback()
		if errR != nil {
			err = fmt.Errorf(MSG_ROLLBACK_ERROR, errR)
		}
		return nil, err
	}

	return rows, nil
}
