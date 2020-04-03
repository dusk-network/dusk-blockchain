package kadcast

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastChunksMsg(t *testing.T) {

	t.SkipNow()

	log.SetLevel(log.TraceLevel)

	// Initiate kadcast network bootstraping of 10 nodes
	nodes := StartKadcastNetwork(10, 9000)

	// Wait until network discovery completes for all nodes
	cond1 := func() bool {
		for _, r := range nodes {
			totalPeersNum := r.tree.getTotalPeers()
			// TODO: Consider why PeersNum is inconsistent
			if int(totalPeersNum) != len(nodes)-1 && int(totalPeersNum) != len(nodes) {
				return false
			}
		}
		return true
	}
	assert.Eventually(t, cond1, 1*time.Minute, 1*time.Second)

	// Broadcast Chunk message
	payload := []byte{1, 2, 3, 4, 7, 8, 9}

	sender := 1
	nodes[sender].StartPacketBroadcast(0, payload)

	time.Sleep(3 * time.Second)

	failed := false
	for i, r := range nodes {

		// Sender of CHUNK message not needed to be tested
		if sender == i {
			continue
		}

		received := false
		for _, chunk := range r.chunkIDmap {
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

// Starting a node. A node is represented by a routing state, TCP listener and
// UDP listener
func startMockNode(port int) *Router {

	log.Infoln("Starting Kadcast Node at :", port)

	p := mockPeer(uint16(port))
	router := makeRouterFromPeer(p)

	// Force each node to store all chunk messages
	// Needed only for testing purposes
	router.storeChunks = true

	// Initialize the UDP server
	udpQueue := ring.NewBuffer(500)
	// Launch PacketProcessor routine.
	go ProcessUDPPacket(udpQueue, &router)
	// Launch a listener routine.
	go StartUDPListener("udp4", udpQueue, router.MyPeerInfo)

	// Initialize the TCP server
	tcpQueue := ring.NewBuffer(500)
	// Launch PacketProcessor routine.
	go ProcessTCPPacket(tcpQueue, &router)
	// Launch a listener routine.
	go StartTCPListener("tcp4", tcpQueue, router.MyPeerInfo)

	return &router
}

func StartKadcastNetwork(num int, basePort int) []*Router {

	// List of all peer routers
	routers := make([]*Router, 0)
	bootstrapNodes := make([]Peer, 0)

	for i := 0; i < num; i++ {
		r := startMockNode(basePort + i)
		bootstrapNodes = append(bootstrapNodes, r.MyPeerInfo)
		routers = append(routers, r)
	}

	// Start Bootstrapping process.
	err := InitBootstrap(routers[0], bootstrapNodes)
	if err != nil {
		log.Panic("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	// TODO: Should we trigger network discovery for each peer
	for _, r := range routers {
		StartNetworkDiscovery(r)
	}

	return routers
}
