package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
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

func testUDPConn(t *testing.T) {
	//go func () {
	lAddr := getLocalUDPAddress()
	pc, err := net.ListenUDP("udp", &lAddr)
	if err != nil {
		log.Println(err)
	}

	//simple read
	buffer := make([]byte, 1024)
	pc.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, _, er := pc.ReadFromUDP(buffer)

	if er != nil {
		log.Printf("%v", er)
	}
	//}()
	for {

	}
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
	go StartUDPListener("udp", queue, router.myPeerInfo)

	/*// Create BootstrapNodes Peer structs
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
	*/
	for {

	}
}
