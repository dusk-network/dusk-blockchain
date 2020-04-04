package kadcast

import (
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/prometheus/common/log"
)

// testPeer creates a peer with local IP and
// ID that is computed over port number but not IP
func testPeer(port uint16) Peer {

	lAddr := getLocalUDPAddress(int(port))
	var ip [4]byte
	copy(ip[:], lAddr.IP)

	// ID is computed over port number but not IP
	var b [4]byte
	b[0] = byte(port)
	b[1] = byte(port >> 8)
	id := computePeerID(b)

	return Peer{ip, port, id}
}

// TestNode starts a node for testing purposes. A node is represented by a
// routing state, TCP listener and UDP listener
func TestNode(port int) *Router {

	log.Infoln("Starting Kadcast Node at :", port)

	peer := testPeer(uint16(port))
	router := makeRouterFromPeer(peer)

	// Force each node to store all chunk messages
	// Needed only for testing purposes
	router.StoreChunks = true

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

// TestNetwork initiates kadcast network bootstraping of N nodes. This will run
// a set of nodes bound on local addresses (port per node), execute
// bootstrapping and network discovery
func TestNetwork(num int, basePort int) ([]*Router, error) {

	// List of all peer routers
	routers := make([]*Router, 0)
	bootstrapNodes := make([]Peer, 0)

	for i := 0; i < num; i++ {
		r := TestNode(basePort + i)
		bootstrapNodes = append(bootstrapNodes, r.MyPeerInfo)
		routers = append(routers, r)
	}

	// Start Bootstrapping process.
	err := InitBootstrap(routers[0], bootstrapNodes)
	if err != nil {
		return nil, err
	}

	// Once the bootstrap succeeded, start the network discovery.
	// TODO: Should we trigger network discovery for each peer
	for _, r := range routers {
		StartNetworkDiscovery(r)
	}

	return routers, nil
}
