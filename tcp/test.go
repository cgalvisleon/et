package tcp

import "fmt"

func test(srv *Server) {
	for _, peer := range srv.raft.peers {
		if peer.Status != Connected {
			err := peer.Connect()
			if err != nil {
				fmt.Println(err)
				continue
			}
		}

		res := peer.Request("Tcp.Ping", srv.addr)
		if res.Error != nil {
			fmt.Println(res.Error)
			return
		}

		var message string
		err := res.Get(&message)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("response:", message)
	}

}
