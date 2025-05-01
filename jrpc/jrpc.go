package jrpc

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

const RPC_KEY = "apigateway-rpc"

var pkg *Package

/**
* load
**/
func Load(name string) (*Package, error) {
	err := cache.Load()
	if err != nil {
		return nil, err
	}

	if pkg != nil {
		return pkg, nil
	}

	err = config.Validate([]string{
		"RPC_PORT",
	})
	if err != nil {
		return nil, err
	}

	host := config.String("RPC_HOST", "localhost")
	port := config.Int("RPC_PORT", 4200)
	name = strs.DaskSpace(name)
	pkg = &Package{
		Name:    name,
		Host:    host,
		Port:    port,
		Solvers: make(map[string]*Solver),
	}

	return pkg, nil
}

/**
* Close
**/
func Close() {
	logs.Log("Rpc", `Shutting down server...`)
}
