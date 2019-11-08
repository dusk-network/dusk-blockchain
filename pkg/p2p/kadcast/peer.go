package kadcast

import (
	"net"
)

// Peer stores the info of a peer which consists on:
// - IP of the peer.
// - Port to connect to it.
// - The ID of the peer.
type Peer struct {
	ip   net.IP
	port uint16
	id   [16]byte
}

// Constructs a Peer with it's fields values as inputs.
func makePeer(ip net.IP, port uint16, id [16]byte) Peer {
	peer := Peer{ip, port, id}
	return peer
}

// Sets the Id sent as parameter as the Peer ID.
func (peer *Peer) addIP(ip net.IP) {
	peer.ip = ip
}

// Sets the Id sent as parameter as the Peer ID.
func (peer *Peer) addID(id [16]byte) {
	peer.id = id
}

// Sets the port sent as parameter as the Peer port.
func (peer *Peer) addPort(port uint16) {
	peer.port = port
}

// Computes the XOR distance between two Peers.
func (me Peer) computePeerDistance(peer Peer) uint16 {
	return idXor(me.id, peer.id)
}
