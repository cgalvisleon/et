package ettp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/console"
)

/**
* Save
* @return error
**/
func (s *Server) Save() error {
	bytes, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if s.debug {
		console.Debug(string(bytes))
	}

	return nil
}

/**
* Load
* @return error
**/
func (s *Server) Load() error {

	return nil
}
