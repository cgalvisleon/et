package tcp

import (
	"net"
	"sync"
)

type Client struct {
	addr      string
	conn      net.Conn
	mu        sync.Mutex
	connected bool
}
