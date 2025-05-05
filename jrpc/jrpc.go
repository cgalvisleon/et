package jrpc

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

const sourceKey = "apigateway-rpc"

var pkg *Package

func LoadTo(name, host string, port int) (*Package, error) {
	if !utility.ValidStr(name, 1, []string{"", ""}) {
		return nil, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(host, 1, []string{"", ""}) {
		return nil, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	if !utility.ValidInt(port, []int{1, 65535}) {
		return nil, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "port")
	}

	err := cache.Load()
	if err != nil {
		return nil, err
	}

	name = strs.DaskSpace(name)
	result := &Package{
		Name:    name,
		Host:    host,
		Port:    port,
		Solvers: make([]*Solver, 0),
	}

	return result, nil
}

/**
* load
**/
func Load(name string) error {
	if pkg != nil {
		return nil
	}

	err := config.Validate([]string{
		"RPC_PORT",
	})
	if err != nil {
		return err
	}

	host := config.String("RPC_HOST", "localhost")
	port := config.Int("RPC_PORT", 4200)
	pkg, err = LoadTo(name, host, port)
	if err != nil {
		return err
	}

	return nil
}

/**
* Start
**/
func Start() error {
	if pkg == nil {
		return logs.Alertm(ERR_PACKAGE_NOT_FOUND)
	}

	go pkg.start()

	return nil
}

/**
* Close
**/
func Close() {
	if pkg != nil {
		pkg.unMount()
	}

	logs.Log("Rpc", `Shutting down server...`)
}
