package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"testing"
)

func TestConv(t *testing.T) {
	var slce []byte = []byte{0, 0}
	peerNum := binary.BigEndian.Uint16(slce)
	fmt.Printf("%v", peerNum)
}
func testPOW(t *testing.T) {
	a := Peer{
		ip:   [4]byte{192, 169, 1, 1},
		port: 25519,
		id:   [16]byte{22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22},
	}

	println(a.computePeerNonce())
}

func testUDP(t *testing.T) {
	var port uint16 = 25519
	ip := [4]byte{31, 4, 243, 248}
	router := makeRouter(ip, port)

	go startUDPListener("udp", &router)
	destIP := [4]byte{167, 71, 11, 218}
	destPeer := makePeer(destIP, port)
	router.tree.addPeer(router.myPeerInfo, destPeer)

	// Add 4 nodes
	peers := [][4]byte{
		[4]byte{1, 1, 1, 1},
		//[4]byte{2, 2, 2, 2},
		//[4]byte{3, 3, 3, 3},
		//[4]byte{4, 4, 4, 4},
		//[4]byte{5, 5, 5, 5},
		//[4]byte{6, 6, 6, 6},
		//[4]byte{7, 7, 7, 7},
		//[4]byte{8, 8, 8, 8},
		//[4]byte{9, 9, 9, 9},
		//[4]byte{10, 10, 10, 10},
		//[4]byte{11, 11, 11, 11},
		//[4]byte{12, 12, 12, 12},
		//[4]byte{13, 13, 13, 13},
		//[4]byte{14, 14, 14, 14},
		//[4]byte{15, 15, 15, 15},
		//[4]byte{16, 16, 16, 16},
		//[4]byte{17, 17, 17, 17},
		//[4]byte{18, 18, 18, 18},
		//[4]byte{19, 19, 19, 19},
		//[4]byte{20, 20, 20, 20},
		//[4]byte{21, 21, 21, 21},
		//[4]byte{22, 22, 22, 22},
		//[4]byte{23, 23, 23, 23},
		//[4]byte{24, 24, 24, 24},
		//[4]byte{25, 25, 25, 25},
	}

	for _, ip := range peers {
		router.tree.addPeer(router.myPeerInfo, makePeer(ip, port))
	}

	// Send PING packet
	router.sendPing(destPeer)
	router.sendFindNodes()
	//router.sendPing(destPeer)

	for {

	}
}
func TestProtocol(t *testing.T) {
	// Our node info.
	var port uint16 = 25519
	ip := [4]byte{62, 57, 180, 247}
	router := makeRouter(ip, port)

	// Launch a listener for our node.
	go startUDPListener("udp", &router)

	// Create BootstrapNodes Peer structs
	var port1 uint16 = 25519
	ip1 := [4]byte{157, 230, 219, 77}
	boot1 := makePeer(ip1, port1)

	var bootstrapNodes []Peer
	bootstrapNodes = append(bootstrapNodes[:], boot1)

	// Start Bootstrapping process.
	err := initBootstrap(&router, bootstrapNodes)
	if err != nil {
		log.Fatal("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	startNetworkDiscovery(&router)

	for {

	}
}
