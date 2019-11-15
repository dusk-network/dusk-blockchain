package kadcast

import "testing"

func TestStuff(t *testing.T) {
	a := [4]byte{255, 255, 255, 255}
	println(*getUintFromBytes(&a))
}
func testPOW(t *testing.T) {
	a := Peer{
		ip:   [4]byte{192, 169, 1, 1},
		port: 25519,
		id:   [16]byte{22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22},
	}

	println(a.computePeerNonce())
}

func TestUDP(t *testing.T) {
	var port uint16 = 25519
	ip := [4]byte{62, 57, 153, 222}
	router := makeRouter(ip, port)
	//lAddr := getLocalIPAddress()

	go startUDPListener("udp", &router)

	destPeer := makePeer(ip, port)

	// Send PING packet
	router.sendPing(destPeer)
	for {

	}
}
