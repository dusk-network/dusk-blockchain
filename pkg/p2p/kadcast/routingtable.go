package kadcast

import (
	"net"
	"sort"
	"sync"
	"time"
)

// Alpha is the number of nodes to which a node will
// ask for new nodes with `FIND_NODES` messages.
const Alpha int = 3

// InitHeight sets the default initial height for a
// broadcast process.
const InitHeight byte = 128

// RoutingTable holds all of the data needed to interact with
// the routing data and also the networking utils.
type RoutingTable struct {

	// number of delegates per bucket
	beta uint8

	// Tree represents the routing structure.
	tree Tree

	// Local peer fields

	lpeerUDPAddr net.UDPAddr
	LpeerInfo    PeerInfo
	// Holds the Nonce that satisfies: `H(ID || Nonce) < Tdiff`.
	localPeerNonce uint32
}

// MakeRoutingTable allows to create a router which holds the peerInfo and
// also the routing tree information.
func MakeRoutingTable(address string) RoutingTable {
	myPeer, _ := MakePeerFromAddr(address)
	return makeRoutingTableFromPeer(myPeer)
}

func makeRoutingTableFromPeer(peer PeerInfo) RoutingTable {
	return RoutingTable{
		tree:           makeTree(),
		lpeerUDPAddr:   peer.getUDPAddr(),
		LpeerInfo:      peer,
		localPeerNonce: peer.computePeerNonce(),
		beta:           DefaultMaxBetaDelegates,
	}
}

// --------------------------------------------------//
//													 //
// Tools to get sorted Peers in respect to a certain //
// PeerID in terms of XOR-distance.				     //
//													 //
// --------------------------------------------------//

// Returns the complete list of Peers in order to be sorted
// as they have the xor distance in respec to a Peer as a parameter.
func (router *RoutingTable) getPeerSortDist(refPeer PeerInfo) []PeerSort {
	var peerList []PeerInfo
	router.tree.mu.RLock()
	for buckIdx, bucket := range router.tree.buckets {
		if buckIdx == 0 {
			// Look for neighbor peer in spanning tree
			for _, p := range bucket.entries {
				if !router.LpeerInfo.IsEqual(p) {
					// neighbor peer
					peerList = append(peerList, p)
				}
			}
		} else {
			peerList = append(peerList, bucket.entries[:]...)
		}
	}
	router.tree.mu.RUnlock()
	var peerListSort []PeerSort
	for _, peer := range peerList {
		// We don't want to return the Peer struct of the Peer
		// that is the reference.
		if peer != refPeer {
			peerListSort = append(peerListSort[:],
				PeerSort{
					ip:        peer.ip,
					port:      peer.port,
					id:        peer.id,
					xorMyPeer: xor(refPeer.id, peer.id),
				})
		}
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
func (router *RoutingTable) getXClosestPeersTo(peerNum int, refPeer PeerInfo) []PeerInfo {
	var xPeers []PeerInfo
	peerList := router.getPeerSortDist(refPeer)
	sort.Sort(ByXORDist(peerList))

	// Get the `peerNum` closest ones.
	for _, peer := range peerList {
		xPeers = append(xPeers[:],
			PeerInfo{
				ip:   peer.ip,
				port: peer.port,
				id:   peer.id,
			})
		if len(xPeers) >= peerNum {
			break
		}
	}
	return xPeers
}

// Sends a `FIND_NODES` messages to the `alpha` closest peers
// the node knows and waits for a certain time in order to wait
// for the `PONG` message arrivals.
// Then looks for the closest peer to the node itself into the
// buckets and returns it.
func (router *RoutingTable) pollClosestPeer(t time.Duration) PeerInfo {
	var wg sync.WaitGroup

	wg.Add(1)
	router.sendFindNodes()

	var p PeerInfo
	timer := time.AfterFunc(t, func() {
		ps := router.getXClosestPeersTo(DefaultAlphaClosestNodes, router.LpeerInfo)
		if len(ps) > 0 {
			p = ps[0]
		}
		wg.Done()
	})

	wg.Wait()
	timer.Stop()
	return p
}

// Sends a `PING` messages to the bootstrap nodes that
// the node knows and waits for a certain time in order to wait
// for the `PONG` message arrivals.
// Returns back the new number of peers the node is connected to.
func (router *RoutingTable) pollBootstrappingNodes(bootNodes []PeerInfo, t time.Duration) uint64 {
	var wg sync.WaitGroup
	var peerNum uint64

	wg.Add(1)

	for _, peer := range bootNodes {
		router.sendPing(peer)
	}

	timer := time.AfterFunc(t, func() {
		peerNum = router.tree.getTotalPeers()
		wg.Done()
	})

	wg.Wait()
	timer.Stop()
	return peerNum
}

// ------- Packet-sending utilities for the Router ------- //

// Builds and sends a `PING` packet
func (router *RoutingTable) sendPing(receiver PeerInfo) {
	// Build empty packet.
	var p Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	p.setHeadersInfo(0, router)

	// Since return values from functions are not addressable, we need to
	// allocate the receiver UDPAddr
	destUDPAddr := receiver.getUDPAddr()
	// Send the packet
	sendUDPPacket(router.lpeerUDPAddr, destUDPAddr, marshalPacket(p))
}

// Builds and sends a `PONG` packet
func (router *RoutingTable) sendPong(receiver PeerInfo) {
	// Build empty packet.
	var p Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	p.setHeadersInfo(1, router)

	// Since return values from functions are not addressable, we need to
	// allocate the receiver UDPAddr
	destUDPAddr := receiver.getUDPAddr()
	// Send the packet
	sendUDPPacket(router.lpeerUDPAddr, destUDPAddr, marshalPacket(p))
}

// Builds and sends a `FIND_NODES` packet.
func (router *RoutingTable) sendFindNodes() {
	// Get `Alpha` closest nodes to me.
	destPeers := router.getXClosestPeersTo(Alpha, router.LpeerInfo)
	// Fill the headers with the type, ID, Nonce and destPort.
	for _, peer := range destPeers {
		// Build the packet
		var p Packet
		p.setHeadersInfo(2, router)
		// We don't need to add the ID to the payload snce we already have
		// it in the headers.
		// Send the packet
		sendUDPPacket(router.lpeerUDPAddr, peer.getUDPAddr(), marshalPacket(p))
	}
}

// Builds and sends a `NODES` packet.
func (router *RoutingTable) sendNodes(receiver PeerInfo) {
	// Build empty packet
	var p Packet
	// Set headers
	p.setHeadersInfo(3, router)
	// Set payload with the `k` peers closest to receiver.
	peersToSend := p.setNodesPayload(router, receiver)
	// If we don't have any peers to announce, we just skip sending
	// the `NODES` messsage.
	if peersToSend == 0 {
		return
	}
	sendUDPPacket(router.lpeerUDPAddr, receiver.getUDPAddr(), marshalPacket(p))
}

// GetTotalPeers the total amount of peers that a `Peer` is connected to
func (router *RoutingTable) GetTotalPeers() uint64 {
	return router.tree.getTotalPeers()
}
