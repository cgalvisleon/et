package atrib

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type TypeAtrib int

const (
	AtribTypeText TypeAtrib = iota
	AtribTypeNumber
	AtribTypeDate
	AtribTypeBoolean
	AtribTypeEnum
	AtribTypeLookup
	AtribTypeLookupMulti
)

type Atrib struct {
	Name         string      `json:"name"`
	Type         TypeAtrib   `json:"type"`
	Definition   et.Json     `json:"definition"`
	DefaultValue interface{} `json:"default_value"`
}

/**
 * NewAtrib
 * @param name string
 * @param typeAtrib TypeAtrib
 * @return *Atrib
 */
func NewAtrib(name string, typeAtrib TypeAtrib) *Atrib {
	return &Atrib{
		Name:         name,
		Type:         typeAtrib,
		Definition:   et.Json{},
		DefaultValue: "",
	}
}

/**
 * Json
 * @return et.Json, error
 */
func (s *Atrib) Json() (et.Json, error) {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
 * Load
 * @param data et.Json
 * @return error
 */
func (s *Atrib) Load(data et.Json) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, s)
}
