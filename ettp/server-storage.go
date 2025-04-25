package ettp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

/**
* Mount
* @param route *Router
**/
func (s *Server) Mount(route *Router) {
	s.UpsetRoute(
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

/**
* saveRouter
* @return error
**/
func (s *Server) saveRouter() error {
	data, err := json.Marshal(s.solvers)
	if err != nil {
		return err
	}

	s.Storage.Data = data
	s.Storage.Save()

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) Load() error {
	path, err := file.MakeFolder("data")
	if err != nil {
		return err
	}

	fileName := strs.Format("%s/%s.dt", path, strs.Lowcase(s.Name))
	data := make([]*Router, 0)
	s.Storage, err = file.NewSyncFile(fileName, data)
	if err != nil {
		return err
	}

	err = s.Storage.Unmarshal(&data)
	if err != nil {
		return err
	}

	for _, route := range data {
		s.Mount(route)
	}

	s.mountHandlerFunc()

	return nil
}

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	return s.saveRouter()
}
