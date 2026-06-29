package ettp

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/cgalvisleon/et/cache"
	v1 "github.com/cgalvisleon/et/ettp/v1"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
)

type Storage struct {
	Solvers map[string]*Solver
	Version string
	Key     string
}

/**
* newStorage
* @param s *Server
* @return *Storage
**/
func newStorage(s *Server) *Storage {
	key := fmt.Sprintf("%s:%s", s.Name, s.Version)
	return &Storage{
		Solvers: s.Solvers,
		Version: s.Version,
		Key:     key,
	}
}

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	storage := newStorage(s)
	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	if cache.IsLoad() {
		cache.Set(storage.Key, string(bt), 0)
	} else {
		path := path.Join("./", "apigateway.json")
		err = file.Save(path, storage)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) load() error {
	var storage *Storage
	key := fmt.Sprintf("%s:%s", s.Name, s.Version)
	if cache.IsLoad() {
		if !cache.Exists(key) {
			return s.migrate()
		}

		storage = newStorage(s)
		bt, err := json.Marshal(storage)
		if err != nil {
			return err
		}

		strs, err := cache.Get(key, string(bt))
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(strs), &storage)
		if err != nil {
			return err
		}
	} else {
		path := path.Join("./", "apigateway.json")
		exists, err := file.LoadOrSave(path, &storage)
		if err != nil {
			return err
		}

		if !exists {
			storage = newStorage(s)
		}
	}

	for _, solver := range storage.Solvers {
		if solver.Kind == TpHandler {
			continue
		}

		s.setSolver(
			solver.Kind,
			solver.Method,
			solver.Path,
			solver.Solver,
			solver.TypeHeader,
			solver.Header,
			solver.ExcludeHeader,
			solver.Version,
			solver.PackageName,
			false,
		)
	}

	return nil
}

/**
* migrate
* @return error
**/
func (s *Server) migrate() error {
	logs.Log("Migrating routes...")
	var old = v1.Storage{}
	bt, err := json.Marshal(old)
	if err != nil {
		return err
	}

	storageBeforeKey := "Apigateway-v0.0.1"
	strs, err := cache.Get(storageBeforeKey, string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strs), &old)
	if err != nil {
		return err
	}

	router := old.Router
	for _, route := range router {
		s.SetRouter(
			route.Method,
			route.Path,
			route.Resolve,
			int(route.TpHeader),
			route.Header,
			route.ExcludeHeader,
			0,
			route.PackageName,
			false,
		)
	}

	if s.debug {
		logs.Log("Routes migrated:", len(s.router))
	}

	if err := s.Save(); err != nil {
		logs.Alertf("Failed to save routes: %s", err.Error())
	}

	return nil
}
