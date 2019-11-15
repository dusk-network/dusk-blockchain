package kadcast

import "net"

import "fmt"

type Packet struct {
	headers [22]byte
}

// The function recieves a Packet and
func processPacket(netw net.Addr, byteNum int, payload []byte) {
	fmt.Println(netw.Network())
	fmt.Println(byteNum)
	fmt.Println(payload[:])
}
