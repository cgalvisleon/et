package ettp

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

/**
* New
* @param name string, config *Config
* @return *Server
**/
func New(name string, config *Config) *Server {
	err := cache.Load()
	if err != nil {
		logs.Fatal(err)
	}

	err = event.Load()
	if err != nil {
		logs.Fatal(err)
	}

	result := NewServer(name, config)

	return result
}
