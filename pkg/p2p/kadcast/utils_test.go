package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

func testConv(t *testing.T) {
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

func TestProtocol(t *testing.T) {
	// Our node info.
	var port uint16 = 25519
	ip := [4]byte{62, 57, 180, 247}
	router := MakeRouter(ip, port)

	// Create buffer.
	queue := ring.NewBuffer(500)

	// Launch PacketProcessor rutine.
	go ProcessPacket(queue, &router)

	// Launch a listener for our node.
	go StartUDPListener("udp", queue, router.MyPeerInfo)

	// Create BootstrapNodes Peer structs
	var port1 uint16 = 25519
	ip1 := [4]byte{157, 230, 219, 77}
	boot1 := MakePeer(ip1, port1)
	var bootstrapNodes []Peer
	bootstrapNodes = append(bootstrapNodes[:], boot1)

	// Start Bootstrapping process.
	err := InitBootstrap(&router, bootstrapNodes)
	if err != nil {
		log.Fatal("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	StartNetworkDiscovery(&router)

	for {
		time.Sleep(time.Second * 15)
	}
}
