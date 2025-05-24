package jrpc

import (
	"encoding/json"
	"slices"

	"github.com/cgalvisleon/et/cache"
)

var sourceKey = "jrpc_storage"

type Storage struct {
	Packages []*Package
}

/**
* Save
* @return error
**/
func (s *Package) save() error {
	var data = &Storage{
		Packages: make([]*Package, 0),
	}

	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}

	storage, err := cache.Get(sourceKey, string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(storage), &data)
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(data.Packages, func(p *Package) bool { return p.Name == s.Name })
	if idx != -1 {
		data.Packages[idx] = s
	} else {
		data.Packages = append(data.Packages, s)
	}

	bt, err = json.Marshal(data)
	if err != nil {
		return err
	}

	cache.Set(sourceKey, string(bt), 0)

	return nil
}

/**
* getPackages
* @return []*Package, error
**/
func getPackages() ([]*Package, error) {
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	var result = make([]*Package, 0)
	var data = &Storage{
		Packages: result,
	}

	bt, err := json.Marshal(data)
	if err != nil {
		return result, err
	}

	storage, err := cache.Get(sourceKey, string(bt))
	if err != nil {
		return result, err
	}

	err = json.Unmarshal([]byte(storage), &data)
	if err != nil {
		return result, err
	}

	return data.Packages, nil
}
