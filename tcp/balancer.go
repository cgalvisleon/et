package tcp

import "sync/atomic"

type Node struct {
	Address string
	Alive   atomic.Bool
	Conns   atomic.Int64
}

/**
* newNode
* @param addr string
* @return *Node
**/
func newNode(addr string) *Node {
	n := &Node{Address: addr}
	n.Alive.Store(true)
	return n
}

type Balancer struct {
	nodes []*Node
	index atomic.Uint64
}

/**
* newBalancer
* @return *Balancer
**/
func newBalancer() *Balancer {
	return &Balancer{
		nodes: make([]*Node, 0),
		index: atomic.Uint64{},
	}
}

/**
* next
* @return *Node
**/
func (b *Balancer) next() *Node {
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
