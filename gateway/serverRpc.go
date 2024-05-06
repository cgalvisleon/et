package gateway

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Service et.Item

func initRpc() error {
	service := new(Service)

	err := rpc.Register(service)
	if err != nil {
		return logs.Error(err)
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
	port := envar.GetInt(0, "RPC")

	result, err := net.Listen("tcp", fmt.Sprintf(`0.0.0.0:%d`, port))
	if err != nil {
		panic(err)
	}

	return result
}
