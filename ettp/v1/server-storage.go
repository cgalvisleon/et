package ettp

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/console"
)

type Storage struct {
	Router  []*Router
	Proxy   map[string]*Proxy
	Version string
}

func NewStorage() *Storage {
	return &Storage{
		Router:  make([]*Router, 0),
		Proxy:   make(map[string]*Proxy),
		Version: "0.0.1",
	}
}

/**
* migrate
* @return error
**/
func (s *Server) migrate() error {
	console.Log("Migrating routes...")
	var storage = []*Router{}
	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	storageBeforeKey := fmt.Sprintf("%s-v0.0.0", s.Name)
	strs, err := cache.Get(storageBeforeKey, string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strs), &storage)
	if err != nil {
		return err
	}

	n := len(storage)
	for i, route := range storage {
		s.setRouter(
			route.Id,
			route.Method,
			route.Path,
			route.Resolve,
			route.Kind,
			route.Header,
			route.TpHeader,
			route.ExcludeHeader,
			route.Private,
			route.PackageName,
			i == n-1,
		)
	}

	storageKeyBackup := fmt.Sprintf("%s-Backup", s.Name)
	s.mountApiGatewayFunc()
	cache.SetW(storageKeyBackup, strs, 3)

	if s.debug {
		console.Log("Routes migrated:", len(s.router))
	}

	if err := s.save(); err != nil {
		console.Alertf("Failed to save routes: %s", err.Error())
	}

	return nil
}

/**
* save
* @return error
**/
func (s *Server) save() error {
	storage := NewStorage()
	storage.Router = s.solvers
	storage.Proxy = s.proxys
	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	cache.Set(s.storageKey, string(bt), 0)

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) load() error {
	if !cache.Exists(s.storageKey) {
		return s.migrate()
	}

	storage := NewStorage()
	bt, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	strs, err := cache.Get(s.storageKey, string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strs), &storage)
	if err != nil {
		return err
	}

	for _, route := range storage.Router {
		s.setRouter(
			route.Id,
			route.Method,
			route.Path,
			route.Resolve,
			route.Kind,
			route.Header,
			route.TpHeader,
			route.ExcludeHeader,
			route.Private,
			route.PackageName,
			false,
		)
	}

	for _, proxy := range storage.Proxy {
		s.mountProxy(proxy)
	}

	s.mountApiGatewayFunc()

	return nil
}

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	return s.save()
}
