package kadcast_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
)

// TestBroadcastChunksMsg boostrap a kadcast network and make an attempt to
// broadcast a message to all network peers
func TestBroadcastChunksMsg(t *testing.T) {

	nodes, err := kadcast.TestNetwork(1000, 9000)
	if err != nil {
		t.Error(err)
	}

	// Broadcast Chunk message
	payload := []byte{1, 2, 3, 4, 7, 8, 9}

	// Node at position 3 tries to broadcast a chunk message with a dummy
	// payload to the network. Expected here is all nodes to receive the CHUNK
	// message as per protocol specification
	sender := 3
	nodes[sender].StartPacketBroadcast(0, payload)

	time.Sleep(3 * time.Second)

	// Verify if all nodes have received the payload
	failed := false
	for i, r := range nodes {

		// Sender of CHUNK message not needed to be tested
		if sender == i {
			continue
		}

		received := false
		for _, chunk := range r.ChunkIDmap {
			if bytes.Equal(chunk, payload) {
				received = true
				break
			}
		}

		if !received {
			t.Errorf("Peer %s did not receive CHUNK message", r.MyPeerInfo.String())
			failed = true
		}
	}

	if failed {
		t.Error("broadcast message did not reach all network nodes")
	}

	for _, r := range nodes {
		kadcast.TraceRoutingState(r)
	}
}

func TestRoutingState(t *testing.T) {
	// Ensure that the routing state of each node is correct
	// TODO:
}
