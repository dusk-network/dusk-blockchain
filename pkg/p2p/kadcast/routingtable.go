// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
)

// Alpha is the number of nodes to which a node will
// ask for new nodes with `FIND_NODES` messages.
const Alpha int = 3

// RoutingTable holds all of the data needed to interact with
// the routing data and also the networking utils.
type RoutingTable struct {
	// Number of delegates per bucket.
	beta uint8

	// Tree represents the routing structure.
	tree Tree

	// Local peer fields.
	lpeerUDPAddr net.UDPAddr
	LpeerInfo    encoding.PeerInfo
	// Holds the Nonce that satisfies: `H(ID || Nonce) < Tdiff`.
	localPeerNonce uint32
}

// MakeRoutingTable allows to create a router which holds the peerInfo and
// also the routing tree information.
func MakeRoutingTable(address string) RoutingTable {
	myPeer, _ := encoding.MakePeerFromAddr(address)
	return makeRoutingTableFromPeer(myPeer)
}

func makeRoutingTableFromPeer(peer encoding.PeerInfo) RoutingTable {
	return RoutingTable{
		tree:           makeTree(),
		lpeerUDPAddr:   peer.GetUDPAddr(),
		LpeerInfo:      peer,
		localPeerNonce: encoding.ComputeNonce(peer.ID[:]),
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
func (rt *RoutingTable) getPeerSortDist(refPeer encoding.PeerInfo) []PeerSort {
	var peerList []encoding.PeerInfo

	rt.tree.mu.RLock()

	for buckIdx, bucket := range rt.tree.buckets {
		if buckIdx == 0 {
			// Look for neighbor peer in spanning tree
			for _, p := range bucket.entries {
				if !rt.LpeerInfo.IsEqual(p) {
					// neighbor peer
					peerList = append(peerList, p)
				}
			}
		} else {
			peerList = append(peerList, bucket.entries[:]...)
		}
	}

	rt.tree.mu.RUnlock()

	var peerListSort []PeerSort

	for _, peer := range peerList {
		// We don't want to return the Peer struct of the Peer
		// that is the reference.
		if peer != refPeer {
			peerListSort = append(peerListSort[:],
				PeerSort{
					ip:        peer.IP,
					port:      peer.Port,
					id:        peer.ID,
					xorMyPeer: xor(refPeer.ID, peer.ID),
				})
		}
	}

	return peerListSort
}

// PeerSort is a helper type to sort `Peers`.
type PeerSort struct {
	ip        [4]byte
	port      uint16
	id        [16]byte
	xorMyPeer [16]byte
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
func (rt *RoutingTable) getXClosestPeersTo(peerNum int, refPeer encoding.PeerInfo) []encoding.PeerInfo {
	peerList := rt.getPeerSortDist(refPeer)
	xPeers := make([]encoding.PeerInfo, len(peerList))

	sort.Sort(ByXORDist(peerList))

	// Get the `peerNum` closest ones.
	for i, peer := range peerList {
		xPeers[i] = encoding.PeerInfo{
			IP:   peer.ip,
			Port: peer.port,
			ID:   peer.id,
		}

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
func (rt *RoutingTable) pollClosestPeer(t time.Duration) encoding.PeerInfo {
	var wg sync.WaitGroup

	wg.Add(1)
	rt.sendFindNodes()

	var p encoding.PeerInfo

	timer := time.AfterFunc(t, func() {
		ps := rt.getXClosestPeersTo(DefaultAlphaClosestNodes, rt.LpeerInfo)
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
func (rt *RoutingTable) pollBootstrappingNodes(bootNodes []encoding.PeerInfo, t time.Duration) uint64 {
	var wg sync.WaitGroup
	var peerNum uint64

	wg.Add(1)

	for _, peer := range bootNodes {
		h := makeHeader(encoding.PingMsg, rt)

		var buf bytes.Buffer
		if err := encoding.MarshalBinary(h, nil, &buf); err != nil {
			continue
		}

		sendUDPPacket(rt.lpeerUDPAddr, peer.GetUDPAddr(), buf.Bytes())
	}

	timer := time.AfterFunc(t, func() {
		peerNum = rt.tree.getTotalPeers()
		wg.Done()
	})

	wg.Wait()
	timer.Stop()
	return peerNum
}

// Builds and sends a `FIND_NODES` packet.
func (rt *RoutingTable) sendFindNodes() {
	// Get `Alpha` closest nodes to me.
	destPeers := rt.getXClosestPeersTo(Alpha, rt.LpeerInfo)
	// Fill the headers with the type, ID, Nonce and destPort.
	for _, peer := range destPeers {
		var buf bytes.Buffer

		h := makeHeader(encoding.FindNodesMsg, rt)
		if err := encoding.MarshalBinary(h, nil, &buf); err != nil {
			return
		}

		sendUDPPacket(rt.lpeerUDPAddr, peer.GetUDPAddr(), buf.Bytes())
	}
}

// GetTotalPeers the total amount of peers that a `Peer` is connected to.
func (rt *RoutingTable) GetTotalPeers() uint64 {
	return rt.tree.getTotalPeers()
}
