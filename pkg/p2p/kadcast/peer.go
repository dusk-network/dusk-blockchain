package kadcast

import (
	"net"
)

/// Peer stores the
type Peer struct {
	ip   net.IP
	port uint
	id   [32]byte
}

func new_peer(ip *net.IP, port *uint, id *[32]byte) Peer {
	panic("Uninplemented!!")
}
