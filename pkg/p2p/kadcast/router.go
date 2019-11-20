package kadcast

import (
	"net"
	"sort"
)

const K int = 20
const alpha int = 3

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

// Tools to get sorted Peers in respect to a certain
// PeerID in terms of XOR-distance.

// Returns the complete list of Peers in order to be sorted
// as they have the xor distance in respec to a Peer as a parameter.
func (router Router) getPeerSortDist(refPeer Peer) []PeerSort {
	var peerList []Peer
	for buckIdx, bucket := range router.tree.buckets {
		// Skip bucket 0
		if buckIdx != 0 {
			peerList = append(peerList[:], bucket.entries[:]...)
		}
	}
	var peerListSort []PeerSort
	for _, peer := range peerList {
		peerListSort = append(peerListSort[:],
			PeerSort{
				ip:        peer.ip,
				port:      peer.port,
				id:        peer.id,
				xorMyPeer: xor(refPeer.id, peer.id),
			})
	}
	return peerListSort
}

// ByXORDist implements sort.Interface based on the IdDistance
// respective to myPeerId.
type ByXORDist []PeerSort

func (a ByXORDist) Len() int { return len(a) }
func (a ByXORDist) Less(i int, j int) bool {
	return !xorIsBigger(a[i].xorMyPeer, a[j].xorMyPeer)
}
func (a ByXORDist) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Returns a list of the selected number of closest peers
// in respect to a certain `Peer`.
func (router Router) getXClosestPeersTo(peerNum int, refPeer Peer) []Peer {
	var xPeers []Peer
	peerList := router.getPeerSortDist(refPeer)
	sort.Sort(ByXORDist(peerList))

	for _, peer := range peerList {
		xPeers = append(xPeers[:],
			Peer{
				ip:   peer.ip,
				port: peer.port,
				id:   peer.id,
			})
	}
	return xPeers
}

// ------- Packet-sending utilities for the Router ------- //

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

func (router Router) sendFindNodes() {
	// Build empty packet-array of `alpha` packets.
	var packets [alpha]Packet
	// Get `alpha` closest nodes to me.
	destPeers := router.getXClosestPeersTo(alpha, router.myPeerInfo)
	// Fill the headers with the type, ID, Nonce and destPort.
	for i := 0; i < alpha; i++ {
		packets[i].setHeadersInfo(2, router, destPeers[i])
		// We don't need to add the ID to the payload snce we already have
		// it in the headers.
		// Send the packet
		sendUDPPacket("udp", destPeers[i].getUDPAddr(), packets[i].asBytes())
	}
}
