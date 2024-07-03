package jrpc

import (
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

func RpcCall(host string, port int, method string, data et.Json) (et.Item, error) {
	var args []byte = data.ToByte()
	var reply *[]byte

	client, err := rpc.DialHTTP("tcp", strs.Format(`%s:%d`, host, port))
	if err != nil {
		return et.Item{}, logs.Alert(err)
	}
	defer client.Close()

	err = client.Call(method, args, &reply)
	if err != nil {
		return et.Item{}, logs.Alert(err)
	}

	result := et.Json{}.ToItem(*reply)

	return result, nil
}
