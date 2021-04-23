// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Peer is a wrapper of all 2 kadcast processing routing.
type Peer struct {
	// dusk node components
	eventBus  *eventbus.EventBus
	gossip    *protocol.Gossip
	dupemap   *dupemap.DupeMap
	processor *peer.MessageProcessor

	// processors
	m *Maintainer
	w *Writer
	r *Reader

	raptorCodeEnabled bool
}

// NewPeer makes a kadcast peer instance.
func NewPeer(eventBus *eventbus.EventBus, g *protocol.Gossip, dp *dupemap.DupeMap, processor *peer.MessageProcessor, raptorCodeEnabled bool) *Peer {
	return &Peer{eventBus: eventBus, gossip: g, dupemap: dp, processor: processor, raptorCodeEnabled: raptorCodeEnabled}
}

// Launch starts kadcast service.
func (p *Peer) Launch(addr string, bootstrapAddrs []string, beta uint8) {
	// Instantiate Kadcast Router
	router := MakeRoutingTable(addr)
	peerInfo := router.LpeerInfo

	if beta > 0 {
		router.beta = beta
	}

	log.WithField("laddr", peerInfo.String()).
		WithField("MaxDelegatesNum", router.beta).
		WithField("Raptor", p.raptorCodeEnabled).
		Infoln("Starting Kadcast Node")

	// Routing table maintainer.
	// Read-write access to Router
	m := NewMaintainer(&router)
	go m.Serve()

	// A writer for Kadcast broadcast messages
	// Read-only access to Router
	w := NewWriter(&router, p.eventBus, p.gossip, p.raptorCodeEnabled)
	go w.Serve()

	if p.raptorCodeEnabled {
		// A reader for Kadcast broadcast messsages
		r := NewRaptorCodeReader(router.LpeerInfo, p.eventBus, p.gossip, p.processor)
		go r.Serve()
	} else {
		r := NewReader(peerInfo, p.eventBus, p.gossip, p.processor)
		go r.Serve()
	}

	// Start Bootstrapping processes
	go JoinNetwork(&router, bootstrapAddrs)
}

// Close terminates peer service.
func (p *Peer) Close() {
	if p.w != nil {
		_ = p.w.Close()
	}

	if p.r != nil {
		_ = p.r.Close()
	}

	if p.m != nil {
		_ = p.m.Close()
	}
}

// JoinNetwork makes attempts to join the network based on the configured bootstrapping nodes.
func JoinNetwork(router *RoutingTable, bootstrapAddrs []string) {
	bootstrapNodes := make([]encoding.PeerInfo, 0)

	for _, addr := range bootstrapAddrs {
		p, _ := encoding.MakePeerFromAddr(addr)
		bootstrapNodes = append(bootstrapNodes, p)
	}

	err := InitBootstrap(router, bootstrapNodes)
	if err != nil {
		log.Error(err)
	}

	// Once the bootstrap succeeded, start the network discovery.
	StartNetworkDiscovery(router, 2*time.Second)
}

// InitBootstrap inits the Bootstrapping process by sending
// a `PING` message to every bootstrapping node repeatedly.
// If it tried 3 or more times and no new `Peers` were added,
// it panics.
// Otherways, it returns `nil` and logs the Number of peers
// the node is connected to at the end of the process.
func InitBootstrap(router *RoutingTable, bootNodes []encoding.PeerInfo) error {
	log.Info("Bootstrapping process started.")

	// Get PeerList ordered by distance so we can compare it
	// after the `PONG` arrivals.
	initPeerNum := router.tree.getTotalPeers()

	for i := 0; i <= 5; i++ {
		actualPeers := router.pollBootstrappingNodes(bootNodes, time.Second*5)
		if actualPeers <= initPeerNum {
			if i == 5 {
				return errors.New("Maximum number of attempts achieved. Please review yor connection settings")
			}

			log.WithField("Retries", i).Warn("Bootstrapping nodes were not added.Trying again..")
		} else {
			break
		}
	}

	log.WithField("connected_nodes", router.tree.getTotalPeers()).Info("Bootstrapping process finished")
	return nil
}

// StartNetworkDiscovery triggers the network discovery process.
// The node basically sends `FIND_NODES` messages to the nodes it
// is currently connected to and evaluates the `Peers` that were added
// on each iteration.
// If the closest peer to ours is the same during two iterations of the
// `FIND_NODES` message, we finish the process logging the ammout of peers
// we are currently connected to.
// Otherways, if the closest Peer on two consecutive iterations changes, we
// keep queriyng the `alpha` closest nodes with `FIND_NODES` messages.
func StartNetworkDiscovery(router *RoutingTable, d time.Duration) {
	// Get closest actual Peer.
	previousClosestArr := router.getXClosestPeersTo(DefaultAlphaClosestNodes, router.LpeerInfo)

	if len(previousClosestArr) == 0 {
		log.WithField("this", router.LpeerInfo.String()).Error("could not find the closest peers")
		return
	}

	/*
		(1) the node looks up the α closest nodes regarding the XOR-metric in its own buckets.
		(2) It queries these α nodes for the ID by sending FIND_NODE messages.
		(3) The queried nodes respond with a set of k nodes they believe to be closest to ID.
		(4) Based on the acquired information, the node builds a new set of closest nodes and
		iteratively repeats steps (1)-(3), until an iteration does not yield any nodes closer
		than the already known ones anymore.
	*/

	previousClosest := previousClosestArr[0]

	// Ask for new peers, wait for `PONG` arrivals and get the
	// new closest `Peer`.
	actualClosest := router.pollClosestPeer(d)

	// Until we don't get a peer closer to our node on each poll,
	// we look for more nodes.
	for actualClosest != previousClosest {
		previousClosest = actualClosest
		actualClosest = router.pollClosestPeer(d)
	}

	log.WithField("peers_connected", router.tree.getTotalPeers()).
		WithField("this", router.LpeerInfo.String()).
		Info("Network Discovery process finished.")
}
