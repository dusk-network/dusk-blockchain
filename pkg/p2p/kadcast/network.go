package kadcast

import (
	log "github.com/sirupsen/logrus"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// StartUDPListener listens infinitely for UDP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func StartUDPListener(netw string, queue *ring.Buffer, MyPeerInfo Peer) {

	lAddr := getLocalUDPAddress()
	// Set listening port.
	lAddr.Port = int(MyPeerInfo.port)
PacketConnCreation:
	// listen to incoming udp packets
	pc, err := net.ListenUDP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}
	// Set initial deadline.
	pc.SetDeadline(time.Now().Add(time.Minute))

	// Instanciate the buffer
	buffer := make([]byte, 1024)
	for {
		// Read UDP packet.
		byteNum, uAddr, err := pc.ReadFromUDP(buffer)

		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			pc.Close()
			goto PacketConnCreation
		} 
		// Set a new deadline for the connection.
		pc.SetDeadline(time.Now().Add(5 * time.Minute))
		// Serialize the packet.
		encodedPack := encodeRedUDPPacket(uint16(byteNum), *uAddr, buffer[0:byteNum])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
	}
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// reciever `Peer` and sends to it a UDP Packet with the payload inside.
func sendUDPPacket(netw string, addr net.UDPAddr, payload []byte) {
	localAddr := getLocalUDPAddress()
	conn, err := net.DialUDP(netw, &localAddr, &addr)
	if err != nil {
		log.WithError(err).Warn("Could not stablish a connection with the dest Peer.")
		return
	}
	defer conn.Close()

	// Simple write
	_, err = conn.Write(payload)
	if err != nil {
		log.WithError(err).Warn("Error while writting to the filedescriptor.")
		return
	} 
}

// StartTCPListener listens infinitely for TCP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func StartTCPListener(netw string, queue *ring.Buffer, MyPeerInfo Peer) {
	lAddr := getLocalTCPAddress()
	// Set listening port.
	lAddr.Port = int(MyPeerInfo.port)
PacketConnCreation:
	// listen to incoming TCP packets
	listener, err := net.ListenTCP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}
	// Set initial deadline.
	listener.SetDeadline(time.Now().Add(time.Minute))

	// Instanciate the buffer
	buffer := make([]byte, 5000000)
	for {
		// Read TCP packet.
		pc, err := listener.AcceptTCP()
		uAddr := pc.RemoteAddr()
		byteNum, err := pc.Read(buffer)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			pc.Close()
			goto PacketConnCreation
		} 
		// Set a new deadline for the connection.
		pc.SetDeadline(time.Now().Add(5 * time.Minute))
		// Serialize the packet.
		encodedPack := encodeRedTCPPacket(uint16(byteNum), uAddr, buffer[0:byteNum])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
	}
}