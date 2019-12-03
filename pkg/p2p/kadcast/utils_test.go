package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
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
