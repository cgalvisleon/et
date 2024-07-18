package jrpc

import (
	"net/rpc"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

func RpcCall(host string, port int, method string, data js.Json) (js.Item, error) {
	var args []byte = data.ToByte()
	var reply *[]byte

	client, err := rpc.DialHTTP("tcp", strs.Format(`%s:%d`, host, port))
	if err != nil {
		return js.Item{}, logs.Alert(err)
	}
	defer client.Close()

	err = client.Call(method, args, &reply)
	if err != nil {
		return js.Item{}, logs.Alert(err)
	}

	result := js.Json{}.ToItem(*reply)

	return result, nil
}
