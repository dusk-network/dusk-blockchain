package node

import (
	"net"
)

// Node will represent a seeder in a network
type Node struct {
	Addr   net.TCPAddr
	PubKey []byte
}

// NewNode will act as a Node constructor
func NewNode(addr net.TCPAddr, pubKey []byte) *Node {
	return &Node{Addr: addr, PubKey: pubKey}
}
