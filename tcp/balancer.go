package tcp

import "sync/atomic"

type Balancer struct {
	nodes []*Node
	index atomic.Uint64
}

func NewBalancer(nodes []*Node) *Balancer {
	return &Balancer{nodes: nodes}
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
