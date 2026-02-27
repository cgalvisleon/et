package sql

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Status string

const (
	Active       Status = "active"
	Archived     Status = "archived"
	Canceled     Status = "canceled"
	OfSystem     Status = "of_system"
	ForDelete    Status = "for_delete"
	Pending      Status = "pending"
	Approved     Status = "approved"
	Rejected     Status = "rejected"
	Failed       Status = "failed"
	Processed    Status = "processed"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
)

type Transaction struct {
	Model   *Model             `json:"from"`
	Command Command            `json:"command"`
	Data    map[string]et.Json `json:"data"`
	Status  Status             `json:"status"`
}

/**
* addData
* @params idx string, data et.Json)
**/
func (s *Transaction) addData(idx string, data et.Json) {
	s.Data[idx] = data
}

/**
* newTransaction: Creates a new Transaction
* @param from *Model, cmd Command, idx string, data et.Json, status Status
* @return *Transaction
**/
func newTransaction(from *Model, cmd Command, idx string, data et.Json, status Status) *Transaction {
	result := &Transaction{
		Model:   from,
		Command: cmd,
		Data:    make(map[string]et.Json),
		Status:  status,
	}
	result.addData(idx, data)
	return result
}

type Tx struct {
	StartedAt    time.Time       `json:"startedAt"`
	LastUpdateAt time.Time       `json:"lastUpdateAt"`
	ID           string          `json:"id"`
	Transactions []*Transaction  `json:"transactions"`
	onChange     func(*Tx) error `json:"-"`
	isDebug      bool            `json:"-"`
}

/**
* GetTx: Returns the Transaction for the session
* @param tx *Tx
* @return (*Tx, bool)
**/
func GetTx(tx *Tx) (*Tx, bool) {
	if tx != nil {
		return tx, false
	}

	now := timezone.Now()
	id := reg.GenULID("transaction")
	tx = &Tx{
		StartedAt:    now,
		LastUpdateAt: now,
		ID:           id,
		Transactions: make([]*Transaction, 0),
	}
	return tx, true
}

/**
* serialize
* @return []byte, error
**/
func (s *Tx) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Tx) ToJson() (et.Json, error) {
	definition, err := s.serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* OnChange
* @params fn func(*Tx) error
**/
func (s *Tx) OnChange(fn func(*Tx) error) {
	s.onChange = fn
}

/**
* Save
* @return error
**/
func (s *Tx) change() error {
	s.LastUpdateAt = timezone.Now()
	if s.isDebug {
		data, err := s.ToJson()
		if err != nil {
			return err
		}
		logs.Debug(data.ToString())
	}

	if s.onChange != nil {
		return s.onChange(s)
	}

	return nil
}

/**
* AddTransaction: Adds data to the Transaction
* @param from *Model, cmd Command, idx string, data et.Json
**/
func (s *Tx) AddTransaction(from *Model, cmd Command, idx string, data et.Json) error {
	id := slices.IndexFunc(s.Transactions, func(t *Transaction) bool { return t.Model.Key() == from.Key() && t.Command == cmd })
	if id == -1 {
		transaction := newTransaction(from, cmd, idx, data, Pending)
		s.Transactions = append(s.Transactions, transaction)
	} else {
		s.Transactions[id].addData(idx, data)
	}
	return s.change()
}

/**
* SetStatus: Sets the status of a transaction
* @param idx int, status Status
* @return error
**/
func (s *Tx) SetStatus(idx int, status Status) error {
	tr := s.Transactions[idx]
	if tr == nil {
		return errors.New(msg.MSG_TRANSACTION_NOT_FOUND)
	}

	tr.Status = status
	s.Transactions[idx] = tr
	return s.change()
}

/**
* getCache: Returns the data for the from
* @param from *Model
* @return []et.Json
**/
func (s *Tx) GetCache(from *Model) []et.Json {
	result := []et.Json{}
	for _, transaction := range s.Transactions {
		if transaction.Model.Key() == from.Key() && transaction.Command != DELETE {
			for _, data := range transaction.Data {
				result = append(result, data)
			}
		}
	}
	return result
}
