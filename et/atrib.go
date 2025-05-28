package et

import (
	"encoding/json"

	"github.com/google/uuid"
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
	Id           string      `json:"id"`
	Name         string      `json:"name"`
	Type         TypeAtrib   `json:"type"`
	Definition   Json        `json:"definition"`
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
		Id:           uuid.NewString(),
		Name:         name,
		Type:         typeAtrib,
		Definition:   Json{},
		DefaultValue: "",
	}
}

/**
 * Json
 * @return Json, error
 */
func (s *Atrib) Json() (Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
 * Load
 * @param data string
 * @return error
 */
func (s *Atrib) Load(scr string) error {
	return json.Unmarshal([]byte(scr), s)
}
