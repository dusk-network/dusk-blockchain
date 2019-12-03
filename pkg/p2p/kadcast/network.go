package kadcast

import (
	"log"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// Listens infinitely for UDP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func StartUDPListener(netw string, queue *ring.Buffer, myPeerInfo Peer, ) {

	lAddr := getLocalUDPAddress()
	// Set listening port.
	lAddr.Port = int(myPeerInfo.port)
PacketConnCreation:
	// listen to incoming udp packets
	pc, err := net.ListenUDP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}
	// Set initial deadline.
	pc.SetReadDeadline(time.Now().Add(time.Minute))

	// Instanciate the buffer
	buffer := make([]byte, 1024)
	for {

		// Read UDP packet.
		byteNum, uAddr, err := pc.ReadFromUDP(buffer)

		if err != nil {
			log.Printf("%v", err)
			pc.Close()
			goto PacketConnCreation
		} else {
			// Set a new deadline for the connection.
			pc.SetReadDeadline(time.Now().Add(2 * time.Minute))
			// Serialize the packet.
			encodedPack := encodeRedPacket(uint16(byteNum), *uAddr, buffer[0:byteNum])
			// Send the packet to the Consumer putting it on the queue.
			queue.Put(encodedPack)
		}
	}
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// reciever `Peer` and sends to it a UDP Packet with the payload inside.
func sendUDPPacket(netw string, addr net.UDPAddr, payload []byte) {
	localAddr := getLocalUDPAddress()
	conn, err := net.DialUDP(netw, &localAddr, &addr)
	if err != nil {
		log.Println(err)
		return
	}

	// Simple write
	written, err := conn.Write(payload)
	if err != nil {
		log.Println(err)
	} else if written == len(payload) {
		log.Printf("Sent %v bytes to %v", written, addr.IP)
	}
	conn.Close()
}
/*
// Listens infinitely for TCP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func startTCPListener(netw string, queue *ring.Buffer, myPeerInfo Peer) {

	lAddr := getLocalTCPAddress()
	// Set listening port.
	lAddr.Port = int(myPeerInfo.port)
PacketConnCreation:
	// listen to incoming udp packets
	pc, err := net.ListenTCP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}

	// Instanciate the buffer
	buffer := make([]byte, 1024)
	for {
		// Accept TCP connection
		conn, err := pc.AcceptTCP()
		// Read UDP packet.
		byteNum, uAddr, err := conn.ReadFrom(buffer)

		if err != nil {
			log.Printf("%v", err)
			pc.Close()
			goto PacketConnCreation
		} else {
			// Set a new deadline for the connection.
			pc.SetReadDeadline(time.Now().Add(3 * time.Minute))
			// Serialize the packet.
			encodedPack := encodeRedPacket(uint16(byteNum), *uAddr, buffer[0:byteNum])
			// Send the packet to the Consumer putting it on the queue.
			queue.Put(encodedPack)
		}
	}
}*/
