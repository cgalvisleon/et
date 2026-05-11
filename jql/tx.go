package jql

import (
	"database/sql"
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
* getTx
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
* begin
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
* setStatus
* @param status TxStatus
**/
func (s *Tx) setStatus(status TxStatus) {
	s.Status = string(status)
	s.LastUpdateAt = timezone.Now()
}

/**
* commit
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
* rollback
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
