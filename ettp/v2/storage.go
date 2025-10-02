package ettp

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/cache"
	v1 "github.com/cgalvisleon/et/ettp/v1"
	"github.com/cgalvisleon/et/logs"
)

type Storage struct {
	Solvers map[string]*Solver
	Version string
	Key     string
}

func NewStorage(s *Server) *Storage {
	return &Storage{
		Solvers: s.Solvers,
		Version: s.Version,
		Key:     fmt.Sprintf("%s-%s", s.Name, s.Version),
	}
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
			route.Private,
			route.PackageName,
			false,
		)
	}

	if s.debug {
		logs.Log("Routes migrated:", len(s.Router))
	}

	if err := s.Save(); err != nil {
		logs.Alertf("Failed to save routes: %s", err.Error())
	}

	return nil
}

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	storage := NewStorage(s)
	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	cache.Set(storage.Key, string(bt), 0)

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) load() error {
	storage := NewStorage(s)
	if !cache.Exists(storage.Key) {
		return s.migrate()
	}

	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	strs, err := cache.Get(storage.Key, string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strs), &storage)
	if err != nil {
		return err
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
			solver.Private,
			solver.PackageName,
			false,
		)
	}

	return nil
}
