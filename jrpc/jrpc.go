package jrpc

import (
	"encoding/gob"
	"fmt"
	"runtime"
	"slices"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

var (
	pkg *Package
	os  = ""
)

func LoadTo(name, host string, port int) (*Package, error) {
	if !utility.ValidStr(name, 1, []string{"", ""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(host, 1, []string{"", ""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	if !utility.ValidInt(port, []int{1, 65535}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "port")
	}

	name = strs.DaskSpace(name)
	result := NewPackage(name, host, port)

	return result, nil
}

/**
* load
**/
func Load(name string) error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil
	}

	if pkg != nil {
		return nil
	}

	err := cache.Load()
	if err != nil {
		return err
	}

	err = config.Validate([]string{
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
	logs.Log("Rpc", `Shutting down server...`)
}

func init() {
	os = runtime.GOOS
	gob.Register(map[string]interface{}{})
	gob.Register(et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
}
