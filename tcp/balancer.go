package tcp

import "sync/atomic"

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
