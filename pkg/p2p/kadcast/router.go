package kadcast

import (
	"net"
)

const K uint = 20

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
	// Build empty packet.
	var packet Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	packet.setHeadersInfo(0, router, reciever)

	// Since return values from functions are not addressable, we need to
	// allocate the reciever UDPAddr
	destUDPAddr := reciever.getUDPAddr()
	// Send the packet
	sendUDPPacket("udp", destUDPAddr, packet.asBytes())
}

// Builds and sends a `PONG` packet
func (router Router) sendPong(reciever Peer) {
	// Build empty packet.
	var packet Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	packet.setHeadersInfo(1, router, reciever)

	// Since return values from functions are not addressable, we need to
	// allocate the reciever UDPAddr
	destUDPAddr := reciever.getUDPAddr()
	// Send the packet
	sendUDPPacket("udp", destUDPAddr, packet.asBytes())
}

func (router Router) sendFindNode(reciever Peer) {
	// Build empty packet.
	var packet Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	packet.setHeadersInfo(2, router, reciever)
	//TODO: Take clear IDtarget
	// Since return values from functions are not addressable, we need to
	// allocate the reciever UDPAddr
	destUDPAddr := reciever.getUDPAddr()
	// Send the packet
	sendUDPPacket("udp", destUDPAddr, packet.asBytes())
}
