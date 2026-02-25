package tcp

import (
	"time"

	"github.com/cgalvisleon/et/logs"
)

func test(srv *Node) {
	var node *Client
	for _, peer := range srv.peers {
		if peer.Status != Connected {
			err := peer.Connect()
			if err != nil {
				logs.Error(err)
				continue
			} else {
				node = peer
			}
		}
	}

	if node == nil {
		return
	}

	send := func() {
		time.Sleep(2 * time.Second)

		rqs := node.Request("Tcp.Ping")
		if rqs.Error != nil {
			logs.Error(rqs.Error)
			return
		}

		var res *Response
		err := rqs.Get(&res)
		if err != nil {
			logs.Error(err)
		} else {
			logs.Debug("response:", res.ToString())
		}
	}

	for {
		send()
	}
}
