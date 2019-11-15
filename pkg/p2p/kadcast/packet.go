package kadcast

import (
	"log"
	"net"
)

type Packet struct {
	headers [22]byte
}

// The function recieves a Packet and
func processPacket(sender net.UDPAddr, byteNum int, payload []byte, router *Router) {
	log.Printf("Revieved new packet from: %s", sender.IP)
	destIP, port := getPeerNetworkInfo(sender)
	var id [16]byte
	copy(id[:], payload[1:17])
	dest := Peer {
		ip: destIP,
		port: port,
		id: id,
	}
	if payload[0] == 0 {
		router.sendPong(dest)
	}
}
