package kadcast

import (
	"net"
)

// Builds a `PING` packet 
func sendPing(laddr *net.UDPAddr, peer Peer)  {
	payload := make([]byte, 21)

	sendUDPPacket("DuskNetwork", laddr, peer.getUDPAddr(), payload)
}
// The function recieves a Packet and 
func processPacket(netw net.Addr, byteNum int, payload []byte)  {
	panic("Not implemented yet")
}