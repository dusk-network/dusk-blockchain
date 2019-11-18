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
	myPeerInfo    Peer
	// Holds the Nonce that satisfies: `H(ID || Nonce) < Tdiff`.
	myPeerNonce uint32
}

func makeRouter(externIP [4]byte, port uint16) Router {
	myPeer := makePeer(externIP, port)
	return Router{
		tree:          makeTree(myPeer),
		myPeerUDPAddr: myPeer.getUDPAddr(),
		myPeerInfo:    myPeer,
		myPeerNonce:   myPeer.computePeerNonce(),
	}
}

// Builds and sends a `PING` packet
func (router Router) sendPing(reciever Peer) {
	// `PING` Type = 0
	packType := [1]byte{0}
	// Build `PING` payload.
	// Attach sender ID
	payload := append(packType[:], router.myPeerInfo.id[:]...)
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.myPeerNonce)
	payload = append(payload[:], idNonce[:]...)
	// Attach Port
	port := getBytesFromUint16(reciever.port)
	payload = append(payload[:], port[:]...)

	// Since return values from functions are not addressable, we need to
	// allocate the reciever UDPAddr
	destUDPAddr := reciever.getUDPAddr()
	// Send the packet
	sendUDPPacket("udp", destUDPAddr, payload)
}

// Builds and sends a `PONG` packet
func (router Router) sendPong(reciever Peer) {
	// `PONG` Type = 1
	packType := [1]byte{1}
	// Build `PING` payload.
	// Attach sender ID
	payload := append(packType[:], router.myPeerInfo.id[:]...)
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.myPeerNonce)
	payload = append(payload[:], idNonce[:]...)
	// Attach Port
	port := getBytesFromUint16(reciever.port)
	payload = append(payload[:], port[:]...)
	
	// Since return values from functions are not addressable, we need to
	// allocate the reciever UDPAddr
	destUDPAddr := reciever.getUDPAddr()
	// Send the packet
	sendUDPPacket("udp", destUDPAddr, payload)
}
