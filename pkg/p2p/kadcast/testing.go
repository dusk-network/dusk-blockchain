// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// TraceRoutingState logs the routing table of a peer.
func TraceRoutingState(r *RoutingTable) {
	peer := r.LpeerInfo

	log.Tracef("this_peer: %s, bucket peers num %d", peer.String(), r.tree.getTotalPeers())

	for _, b := range r.tree.buckets {
		for _, p := range b.entries {
			_, dist := ComputeDistance(peer, p)

			log.Tracef("bucket: %d, peer: %s, distance: %s", b.idLength, p.String(), hex.EncodeToString(dist[:]))
		}
	}
}

// testPeerInfo creates a peer with local IP.
func testPeerInfo(port uint16) encoding.PeerInfo {
	lAddr := getLocalUDPAddress(int(port))

	var ip [4]byte
	copy(ip[:], lAddr.IP)

	peer := encoding.MakePeer(ip, port)
	return peer
}

// TestRouter creates a peer router with a fixed id.
func TestRouter(port uint16, id [16]byte) *RoutingTable {
	lAddr := getLocalUDPAddress(int(port))

	var ip [4]byte
	copy(ip[:], lAddr.IP)

	peer := encoding.PeerInfo{
		IP:   ip,
		Port: port,
		ID:   id,
	}

	r := makeRoutingTableFromPeer(peer)
	return &r
}

// Node a very limited instance of a node running kadcast peer only.
type Node struct {
	Router   *RoutingTable
	EventBus *eventbus.EventBus

	Lock       sync.RWMutex
	Blocks     []*block.Block
	Duplicated bool
}

func (n *Node) onAcceptBlock(m message.Message) {
	n.Lock.Lock()
	defer n.Lock.Unlock()

	blk := m.Payload().(block.Block)
	n.Blocks = append(n.Blocks, &blk)

	height := m.Header()[0]
	if height > 128 {
		panic("invalid kadcast height")
	}

	/// repropagate
	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, &blk); err != nil {
		panic(err)
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		panic(err)
	}

	repropagateMsg := message.NewWithHeader(topics.Block, *buf, m.Header())
	n.EventBus.Publish(topics.Kadcast, repropagateMsg)
}

func newKadcastNode(r *RoutingTable, eb *eventbus.EventBus) *Node {
	n := &Node{
		Router:   r,
		EventBus: eb,
		Blocks:   make([]*block.Block, 0),
	}

	// Subscribe for topics.Block so we can ensure the kadcast-ed block has been
	// successfully published to the bus
	cbListener := eventbus.NewCallbackListener(n.onAcceptBlock)
	eb.Subscribe(topics.Block, cbListener)

	return n
}

// TestNode starts a node for testing purposes. A node is represented by a
// routing state, TCP listener and UDP listener.
func TestNode(port int) *Node {
	eb := eventbus.New()
	g := protocol.NewGossip(protocol.TestNet)
	d := dupemap.NewDupeMap(1, 10000)

	// Instantiate Kadcast Router
	peer := testPeerInfo(uint16(port))
	router := makeRoutingTableFromPeer(peer)

	// Set beta delegates to 1 so we can expect only one message per node
	// optimal broadcast (no duplicates)
	router.beta = 1

	raptorEnabled := true
	log.Infof("Starting Kadcast Node (raptor:%v) on: %s", raptorEnabled, peer.String())

	// Routing table maintainer
	m := NewMaintainer(&router)
	go m.Serve()

	// A reader for Kadcast broadcast messsage.
	//
	// It listens for a valid kadcast wire messages
	//
	// On a new kadcast message ...
	//
	// Reader forwards the message to the eventbus
	// Reader repropagates any valid kadcast wire messages

	// TODO: messageProcessor

	if raptorEnabled {
		r := NewRaptorCodeReader(router.LpeerInfo, eb, g, d, nil)
		go r.Serve()
	} else {
		r := NewReader(router.LpeerInfo, eb, g, d, nil)
		go r.Serve()
	}

	w := NewWriter(&router, eb, g, raptorEnabled)
	go w.Serve()

	return newKadcastNode(&router, eb)
}

// TestNetwork initiates kadcast network bootstraping of N nodes. This will run
// a set of nodes bound on local addresses (port per node), execute
// bootstrapping and network discovery.
func TestNetwork(num int, basePort int) ([]*Node, error) {
	// List of all peer routers
	nodes := make([]*Node, 0)
	bootstrapNodes := make([]encoding.PeerInfo, 0)

	for i := 0; i < num; i++ {
		n := TestNode(basePort + i)

		if i != 0 {
			bootstrapNodes = append(bootstrapNodes, n.Router.LpeerInfo)
		}

		nodes = append(nodes, n)
	}

	// Give all peers a second to start their udp/tcp listeners
	time.Sleep(1 * time.Second)

	// Start Bootstrapping process.
	err := InitBootstrap(nodes[0].Router, bootstrapNodes)
	if err != nil {
		return nil, err
	}

	// Once the bootstrap succeeded, start the network discovery.
	for _, n := range nodes {
		go StartNetworkDiscovery(n.Router, 1*time.Second)
	}

	time.Sleep(10 * time.Second)

	return nodes, nil
}

// DidNetworkReceivedMsg checks if all network nodes have received the
// msgPayload only once.
func DidNetworkReceivedMsg(nodes []*Node, senderIndex int, sentBlock *block.Block) bool {
	failed := false

	for i, n := range nodes {
		// Sender of CHUNK message not needed to be tested
		if senderIndex == i {
			continue
		}

		received := false
		duplicated := false

		n.Lock.RLock()

		for _, b := range n.Blocks {
			if b.Equals(sentBlock) {
				if received {
					duplicated = true
				}

				received = true
			}
		}

		n.Lock.RUnlock()

		if !received {
			failed = true
			// t.Logf("Peer %s did not receive CHUNK message", n.Router.LpeerInfo.String())
			break
		}

		if duplicated {
			failed = true
			// t.Logf("Peer %s receive same CHUNK message more than once", n.Router.LpeerInfo.String())
			break
		}
	}

	return !failed
}

// TestReceivedMsgOnce check periodically (up to 7 sec) if network has received the message.
func TestReceivedMsgOnce(t *testing.T, nodes []*Node, i int, blk *block.Block) {
	passed := false

	for y := 0; y < 200; y++ {
		// Wait a bit so all nodes receives the broadcast message
		time.Sleep(100 * time.Millisecond)

		if DidNetworkReceivedMsg(nodes, i, blk) {
			passed = true
			break
		}
	}

	if passed {
		log.Infof("Each network node received the message once")
	} else {
		log.Fatal("Broadcast procedure has failed")
	}
}
