package kadcast

import (
	"net"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Alpha is the number of nodes to which a node will
// ask for new nodes with `FIND_NODES` messages.
const Alpha int = 3

// InitHeight sets the default initial height for a
// broadcast process.
const InitHeight byte = 128

// Router holds all of the data needed to interact with
// the routing data and also the networking utils.
type Router struct {

	// number of delegates per bucket
	beta uint8

	// Tree represents the routing structure.
	tree Tree
	// Even the port and the IP are the same info, the difference
	// is that one IP has type `IP` and the other `[4]byte`.
	// Since we only store one tree on the application, it's worth
	// to keep both in order to avoid convert the types continuously.
	myPeerUDPAddr net.UDPAddr
	MyPeerInfo    Peer
	// Holds the Nonce that satisfies: `H(ID || Nonce) < Tdiff`.
	myPeerNonce uint32

	// Store all received Chunk payload. Useful on testing.
	// Will be removed when kadcast is integrated with eventbus
	StoreChunks bool
	MapMutex    sync.RWMutex
	ChunkIDmap  map[[16]byte][]byte
	Duplicated  bool
}

// MakeRouter allows to create a router which holds the peerInfo and
// also the routing tree information.
func MakeRouter(externIP [4]byte, port uint16) Router {
	myPeer := MakePeer(externIP, port)
	return makeRouterFromPeer(myPeer)
}

func makeRouterFromPeer(peer Peer) Router {
	return Router{
		tree:          makeTree(),
		myPeerUDPAddr: peer.getUDPAddr(),
		MyPeerInfo:    peer,
		myPeerNonce:   peer.computePeerNonce(),
		ChunkIDmap:    make(map[[16]byte][]byte),
		beta:          DefaultMaxBetaDelegates,
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
func (router *Router) getPeerSortDist(refPeer Peer) []PeerSort {
	var peerList []Peer
	//TODO: #603
	router.tree.mu.RLock()
	for buckIdx, bucket := range router.tree.buckets {
		if buckIdx == 0 {
			// Look for neighbor peer in spanning tree
			for _, p := range bucket.entries {
				if !router.MyPeerInfo.IsEqual(p) {
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
func (router *Router) getXClosestPeersTo(peerNum int, refPeer Peer) []Peer {
	var xPeers []Peer
	peerList := router.getPeerSortDist(refPeer)
	sort.Sort(ByXORDist(peerList))

	// Get the `peerNum` closest ones.
	for _, peer := range peerList {
		xPeers = append(xPeers[:],
			Peer{
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
func (router *Router) pollClosestPeer(t time.Duration) Peer {
	var wg sync.WaitGroup

	wg.Add(1)
	router.sendFindNodes()

	var p Peer
	timer := time.AfterFunc(t, func() {
		ps := router.getXClosestPeersTo(DefaultAlphaClosestNodes, router.MyPeerInfo)
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
func (router *Router) pollBootstrappingNodes(bootNodes []Peer, t time.Duration) uint64 {
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
func (router *Router) sendPing(receiver Peer) {
	// Build empty packet.
	var p Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	p.setHeadersInfo(0, router)

	// Since return values from functions are not addressable, we need to
	// allocate the receiver UDPAddr
	destUDPAddr := receiver.getUDPAddr()
	// Send the packet
	sendUDPPacket(router.myPeerUDPAddr, destUDPAddr, marshalPacket(p))
}

// Builds and sends a `PONG` packet
func (router *Router) sendPong(receiver Peer) {
	// Build empty packet.
	var p Packet
	// Fill the headers with the type, ID, Nonce and destPort.
	p.setHeadersInfo(1, router)

	// Since return values from functions are not addressable, we need to
	// allocate the receiver UDPAddr
	destUDPAddr := receiver.getUDPAddr()
	// Send the packet
	sendUDPPacket(router.myPeerUDPAddr, destUDPAddr, marshalPacket(p))
}

// Builds and sends a `FIND_NODES` packet.
func (router *Router) sendFindNodes() {
	// Get `Alpha` closest nodes to me.
	destPeers := router.getXClosestPeersTo(Alpha, router.MyPeerInfo)
	// Fill the headers with the type, ID, Nonce and destPort.
	for _, peer := range destPeers {
		// Build the packet
		var p Packet
		p.setHeadersInfo(2, router)
		// We don't need to add the ID to the payload snce we already have
		// it in the headers.
		// Send the packet
		sendUDPPacket(router.myPeerUDPAddr, peer.getUDPAddr(), marshalPacket(p))
	}
}

// Builds and sends a `NODES` packet.
func (router *Router) sendNodes(receiver Peer) {
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
	sendUDPPacket(router.myPeerUDPAddr, receiver.getUDPAddr(), marshalPacket(p))
}

// BroadcastPacket sends a `CHUNKS` message across the network
// following the Kadcast broadcasting rules with the specified height.
func (router *Router) broadcastPacket(height byte, tipus byte, payload []byte) {

	if height > byte(len(router.tree.buckets)) || height == 0 {
		return
	}

	myPeer := router.MyPeerInfo

	var i byte
	for i = 0; i <= height-1; i++ {

		// this should be always a deep copy of a bucket from the tree
		var b bucket
		router.tree.mu.RLock()
		b = router.tree.buckets[i]
		router.tree.mu.RUnlock()

		if len(b.entries) == 0 {
			continue
		}

		delegates := make([]Peer, 0)

		if b.idLength == 0 {
			// the bucket B 0 only holds one specific node of distance one
			for _, p := range b.entries {
				// Find neighbor peer
				if !router.MyPeerInfo.IsEqual(p) {
					delegates = append(delegates, p)
					break
				}
			}
		} else {
			/*
				Instead of having a single delegate per bucket, we select β delegates.
				This severely increases the probability that at least one out of the
				multiple selected nodes is honest and reachable.
			*/

			in := make([]Peer, len(b.entries))
			copy(in, b.entries)

			if err := generateRandomDelegates(router.beta, in, &delegates); err != nil {
				log.WithError(err).Warn("generate random delegates failed")
			}
		}

		// For each of the delegates found from this bucket, make an attempt to
		// repropagate CHUNK message
		for _, destPeer := range delegates {

			if myPeer.IsEqual(destPeer) {
				log.Error("Destination peer must be different from the source peer")
				continue
			}

			log.WithField("from", myPeer.String()).
				WithField("to", destPeer.String()).
				WithField("Height", height).
				Trace("Sending CHUNKS message")

			// Create empty packet and set headers.
			var p Packet
			// Set headers info.
			p.setHeadersInfo(tipus, router)
			// Set payload.
			p.setChunksPayloadInfo(i, payload)
			sendTCPStream(destPeer.getUDPAddr(), marshalPacket(p))
		}
	}
}

// StartPacketBroadcast sends a `CHUNKS` message across the network
// following the Kadcast broadcasting rules with the InitHeight.
func (router *Router) StartPacketBroadcast(payload []byte) {
	router.broadcastPacket(InitHeight, 0, payload)
}

// GetTotalPeers the total amount of peers that a `Peer` is connected to
func (router *Router) GetTotalPeers() uint64 {
	return router.tree.getTotalPeers()
}
