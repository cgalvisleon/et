package instances

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

type Model struct {
	model *jsql.Model
}

/**
* LoadModel
* @param model *jsql.Model
* @return *Model
**/
func LoadModel(model *jsql.Model) *Model {
	return &Model{
		model: model,
	}
}

/**
* Set
* @param id, tag, ownerId string, obj any
* @return error
**/
func (s *Model) Set(id, tag, ownerId string, obj any) error {
	bt, err := json.Marshal(s)
	if err != nil {
		return err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return err
	}

	result.Set("id", id)
	result.Set("tag", tag)
	result.Set("owner_id", ownerId)
	_, err = s.model.
		Upsert(result).
		Where(jsql.Eq("id", id)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* Get
* @param id string, dest any
* @return (bool, error)
**/
func (s *Model) Get(id string, dest any) (bool, error) {
	item, err := s.model.
		Where(jsql.Eq("id", id)).
		One()
	if err != nil {
		return false, err
	}

	if !item.Ok {
		return false, nil
	}

	data := []byte(item.Result.ToString())
	err = json.Unmarshal(data, dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* Delete
* @param id string
* @return error
**/
func (s *Model) Delete(id string) error {
	_, err := s.model.
		Delete().
		Where(jsql.Eq("id", id)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
