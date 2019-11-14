package kadcast

import (
	"net"
)
// Router holds all of the data needed to interact with
// the routing data and also the networking utils.
type Router struct {
	// Tree represents the routing structure.
	tree Tree
	// Even the port and the IP are the same info, the difference
	// is that one IP has type `IP` and the other `[4]byte`.
	// Since we only store one tree on the application, it's worth
	// to keep both in order to avoid convert the types continuously.
	myPeerUDPAddr net.UDPAddr
	myPeerInfo Peer
	// Holds the Nonce that satisfies: `H(ID || Nonce) < Tdiff`.
	myPeerNonce uint32
}



// Builds a `PING` packet 
func (router *Router) sendPing(reciever *Peer)  {
	// `PING` Type = 0
	packType := [1]byte{0}
	// Build `PING` payload.
	payload := append(packType[:], router.myPeerInfo.id[:]...)
	payload = append(payload[:], getBytesFromUint(&router.myPeerNonce)[:]...)

	sendUDPPacket("DuskNetwork", &router.myPeerUDPAddr, reciever.getUDPAddr(), payload)
}
// The function recieves a Packet and 
func processPacket(netw net.Addr, byteNum int, payload []byte)  {
	panic("Not implemented yet")
}