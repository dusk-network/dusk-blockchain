package kadcast

import (
	"log"
	"net"
)

// Listens infinitely for UDP packet arrivals and 
// executes it's processing inside a gorutine.
func startUDPListener(netw string, router *Router) {
	lAddr := getLocalIPAddress()
	for {
		// listen to incoming udp packets
		pc, err := net.ListenUDP(netw, &lAddr)
		if err != nil {
			log.Println(err)
		}
		defer pc.Close()

		//simple read
		buffer := make([]byte, 1024)
		
		byteNum, uAddr, _ := pc.ReadFromUDP(buffer)
		go processPacket(*uAddr, byteNum, buffer, router)
	}
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// reciever `Peer` and sends to it a UDP Packet with the payload inside.
func sendUDPPacket(netw string, addr net.UDPAddr, payload []byte) {
	localAddr := getLocalIPAddress()
	conn, err := net.DialUDP(netw, &localAddr, &addr)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	// Simple write
	over, err := conn.Write(payload)
	if err != nil {
		log.Println(err)
	} else if over > 0 {
		log.Printf("UDP_Packet Sending error: %v bytes were not included", over)
	}
}


