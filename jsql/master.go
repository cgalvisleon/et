package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/et"
)

type Master struct {
	From   *From             `json:"from"`
	To     *From             `json:"to"`
	Bridge *From             `json:"bridge"`
	Keys   map[string]string `json:"keys"`
	ToKeys map[string]string `json:"to_keys"`
	Select []string          `json:"select"`
	Rows   int               `json:"rows"`
}

/**
* ref: Returns the reference of the master.
* @return et.Json
**/
func (s *Master) ref() et.Json {
	return et.Json{
		"to":     s.To,
		"bridge": s.Bridge,
	}
}

/**
* init: Initializes the master.
* @return error
**/
func (s *Master) init() error {
	if s.To == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if s.To.Model == nil {
		return errors.New(MSG_TO_MODEL_REQUIRED)
	}

	if s.Bridge == nil {
		return errors.New(MSG_BRIDGE_MODEL_REQUIRED)
	}

	if s.Bridge.Model == nil {
		return errors.New(MSG_BRIDGE_MODEL_REQUIRED)
	}

	// The bridge has foreign keys to both models, so both tables must exist first.
	if s.From != nil && s.From.Model != nil {
		if err := s.From.Model.Init(); err != nil {
			return err
		}
	}

	err := s.To.Model.Init()
	if err != nil {
		return err
	}

	err = s.Bridge.Model.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* newMaster: Creates a new master.
* @param from, to, bridge *Model, keys, toKeys map[string]string, selects []string
* @return *Master
**/
func newMaster(from, to, bridge *Model, keys, toKeys map[string]string, selects []string) *Master {
	return &Master{
		From:   getFrom(from, ""),
		To:     getFrom(to, ""),
		Bridge: getFrom(bridge, ""),
		Keys:   keys,
		ToKeys: toKeys,
		Select: selects,
	}
}
