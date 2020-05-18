package kadcast_test

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

const (
	// basePort all listeners derive from
	basePort = 10000
	// Number of nodes to bootstrap per a test
	networkSize = 10
)

// TestBroadcastChunksMsg boostrap a kadcast network and make an attempt to
// broadcast a message to all network peers
func TestBroadcastChunksMsg(t *testing.T) {

	nodes, err := kadcast.TestNetwork(networkSize, basePort)
	if err != nil {
		t.Error(err)
	}

	// 	log.SetLevel(log.TraceLevel)
	for _, r := range nodes {
		kadcast.TraceRoutingState(r)
	}

	// Broadcast Chunk message. Each of the nodes makes an attempt to broadcast
	// a CHUNK message to the network
	for i := 0; i < len(nodes); i++ {

		payload, _ := crypto.RandEntropy(128)
		//log.WithField("from_node", i).Infof("Broadcasting a message")

		// Node at position i tries to broadcast a chunk message with a random
		// payload to the network. Expected here is all nodes to receive the
		// CHUNK message as per protocol specification without any duplicates
		nodes[i].StartPacketBroadcast(payload)

		/*
			If we assume constant transmission times, honest network partici-
			pants, and no packet loss in the underlying network, the propaga-
			tion method just discussed would result in an optimal broadcast
			tree. In this scenario, every node receives the block exactly once and
			hence no duplicate messages would be induced by this broadcast-
			ing operation.
		*/

		time.Sleep(2 * time.Second)
		kadcast.TestReceivedChunckOnce(t, nodes, i, payload)
		//log.Infof("Each network node received the message once")
	}
}

func TestRoutingState(t *testing.T) {
	// Ensure that the routing state of each node is correct TODO:
}
