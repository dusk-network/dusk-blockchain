package kadcast

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func testConv(t *testing.T) {
	var a uint32 = 4795984
	c := getBytesFromUint32(a)
	fmt.Printf("%v", c)
	b := binary.LittleEndian.Uint32(c[:])
	println(b)
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
	ip := [4]byte{192, 168, 0, 11}
	router := makeRouter(ip, port)

	go startUDPListener("udp", &router)
	destIP := [4]byte{178, 62, 17, 199}
	destPeer := makePeer(destIP, port)

	// Send PING packet
	router.sendPing(destPeer)
	for {

	}
}
