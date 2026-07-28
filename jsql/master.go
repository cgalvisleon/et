package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/et"
)

type Master struct {
	To     *From
	Bridge *From
	Keys   map[string]string
	ToKeys map[string]string
}

/**
* Ref: Returns the reference of the master.
* @return et.Json
**/
func (s *Master) Ref() et.Json {
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
* @param to, bridge *Model, keys, toKeys map[string]string
* @return *Master
**/
func newMaster(to, bridge *Model, keys, toKeys map[string]string) *Master {
	return &Master{
		To:     getFrom(to, ""),
		Bridge: getFrom(bridge, ""),
		Keys:   keys,
		ToKeys: toKeys,
	}
}
