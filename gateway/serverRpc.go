package gateway

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/et"
)

type Service et.Item

func initRpc() error {
	service := new(Service)

	err := rpc.Register(service)
	if err != nil {
		return console.Error(err)
	}

	return nil
}

func (c *Service) Version(rq []byte, rp *et.Item) error {
	result := et.Item{
		Ok:     true,
		Result: Version(),
	}

	*rp = result

	return nil
}

func newRpc() net.Listener {
	initRpc()
	rpc.HandleHTTP()
	port := envar.EnvarInt(0, "RPC")

	result, err := net.Listen("tcp", fmt.Sprintf(`0.0.0.0:%d`, port))
	if err != nil {
		panic(err)
	}

	return result
}
