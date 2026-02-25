package tcp

import (
	"io"
	"net"
	"sync/atomic"
	"time"
)

type node struct {
	Address string
	Alive   atomic.Bool
	Conns   atomic.Int64
}

/**
* newNode
* @param addr string
* @return *node
**/
func newNode(addr string) *node {
	n := &node{Address: addr}
	n.Alive.Store(true)
	return n
}

type Balancer struct {
	nodes []*node
	index atomic.Uint64
}

var proxy *Balancer

/**
* newBalancer
* @return *Balancer
**/
func newBalancer() *Balancer {
	return &Balancer{
		nodes: make([]*node, 0),
		index: atomic.Uint64{},
	}
}

/**
* handleBalancer
* @param client net.Conn
**/
func handleBalancer(client net.Conn) {
	defer client.Close()

	if proxy == nil {
		proxy = newBalancer()
	}

	node := proxy.next()
	if node == nil {
		return
	}

	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	backend, err := dialer.Dial("tcp", node.Address)
	if err != nil {
		return
	}
	defer backend.Close()

	node.Conns.Add(1)
	defer node.Conns.Add(-1)

	go io.Copy(backend, client)
	io.Copy(client, backend)
}

/**
* next
* @return *node
**/
func (b *Balancer) next() *node {
	n := uint64(len(b.nodes))
	for i := uint64(0); i < n; i++ {
		idx := (b.index.Add(1)) % n
		node := b.nodes[idx]
		if node.Alive.Load() {
			return node
		}
	}
	return nil
}
