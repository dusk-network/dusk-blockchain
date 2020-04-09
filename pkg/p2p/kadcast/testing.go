package kadcast

import (
	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	log "github.com/sirupsen/logrus"
)

// TraceRoutingState logs the routing table of a peer
func TraceRoutingState(r *Router) {
	peer := r.MyPeerInfo
	log.Tracef("this_peer: %s, bucket peers num %d", peer.String(), r.tree.getTotalPeers())
	for bucketID, b := range r.tree.buckets {
		if bucketID == 0 {
			continue
		}

		for _, p := range b.entries {
			_, dist := peer.computeDistance(p)
			log.Tracef("bucket: %d, peer: %s, distance: %s", b.idLength, p.String(), hex.EncodeToString(dist[:]))
		}
	}
}

// testPeer creates a peer with local IP
func testPeer(port uint16) Peer {

	lAddr := getLocalUDPAddress(int(port))
	var ip [4]byte
	copy(ip[:], lAddr.IP)

	peer := MakePeer(ip, port)
	return peer
}

// TestNode starts a node for testing purposes. A node is represented by a
// routing state, TCP listener and UDP listener
func TestNode(port int) *Router {

	peer := testPeer(uint16(port))
	router := makeRouterFromPeer(peer)

	log.Infof("Starting Kadcast Node on: %s", peer.String())

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
	for _, r := range routers {
		StartNetworkDiscovery(r)
	}

	return routers, nil
}
