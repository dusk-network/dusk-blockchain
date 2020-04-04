package kadcast_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastChunksMsg(t *testing.T) {

	// Currently test is disabled until kadcast gets stable
	t.SkipNow()

	log.SetLevel(log.TraceLevel)
	nodes, err := kadcast.TestNetwork(10, 9000)
	if err != nil {
		t.Error(err)
	}

	// Wait until network discovery completes for all nodes
	cond1 := func() bool {
		for _, r := range nodes {
			totalPeersNum := r.GetTotalPeers()
			// TODO: Consider why PeersNum is inconsistent
			if int(totalPeersNum) != len(nodes)-1 && int(totalPeersNum) != len(nodes) {
				log.Errorf("not all peers found")
				return false
			}
		}
		return true
	}
	assert.Eventually(t, cond1, 1*time.Minute, 1*time.Second)

	// Broadcast Chunk message
	payload := []byte{1, 2, 3, 4, 7, 8, 9}

	// Node at position 1 tries to broadcast a chunk message with a dummy
	// payload to the network. Expected here is all nodes to receive the CHUNK
	// message as per protocol specification
	sender := 1
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
			t.Logf("Peer %s did not receive CHUNK message", r.MyPeerInfo.String())
			failed = true
		}
	}

	if failed {
		t.Error("not all peers received CHUNK message")
	}
}

func TestRoutingState(t *testing.T) {
	// Ensure that the routing state of each node is correct
	// TODO:
}
