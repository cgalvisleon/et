package jrpc

import (
	"github.com/celsiainternet/elvis/cache"
	"github.com/celsiainternet/elvis/envar"
	"github.com/celsiainternet/elvis/logs"
	"github.com/celsiainternet/elvis/strs"
)

const RPC_KEY = "apigateway-rpc"

var pkg *Package

/**
* load
**/
func Load(name string) (*Package, error) {
	_, err := cache.Load()
	if err != nil {
		return nil, err
	}

	if pkg != nil {
		return pkg, nil
	}

	host := envar.GetStr("localhost", "HOST")
	host = envar.GetStr(host, "RPC_HOST")
	port := envar.GetInt(4200, "RPC_PORT")
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
	if pkg != nil {
		UnMount()
	}

	logs.Log("Rpc", `Shutting down server...`)
}
