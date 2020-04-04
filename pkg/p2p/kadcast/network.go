package kadcast

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

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
	_ = pc.SetDeadline(time.Now().Add(time.Minute))

	// Instantiate the buffer
	buffer := make([]byte, 1024)
	for {
		// Read UDP packet.
		byteNum, uAddr, err := pc.ReadFromUDP(buffer)

		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			_ = pc.Close()
			goto PacketConnCreation
		}
		// Set a new deadline for the connection.
		_ = pc.SetDeadline(time.Now().Add(5 * time.Minute))
		// Serialize the packet.
		encodedPack := encodeReadUDPPacket(uint16(byteNum), *uAddr, buffer[0:byteNum])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
	}
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// receiver `Peer` and sends to it a UDP Packet with the payload inside.
//nolint:unparam
func sendUDPPacket(netw string, addr net.UDPAddr, payload []byte) {
	localAddr := getLocalUDPAddress()
	conn, err := net.DialUDP(netw, &localAddr, &addr)
	if err != nil {
		log.WithError(err).Warn("Could not stablish a connection with the dest Peer.")
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// Simple write
	_, err = conn.Write(payload)
	if err != nil {
		log.WithError(err).Warn("Error while witting to the file descriptor.")
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

	for {
		_ = listener.SetDeadline(time.Now().Add(time.Minute))
		pc, err := listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Warn("Error on tcp accept")
			goto PacketConnCreation
		}

		// Read frame payload
		// Set a new deadline for the connection.
		_ = pc.SetDeadline(time.Now().Add(time.Minute))

		payload, byteNum, err := readTCPFrame(pc)
		if err != nil {
			log.WithError(err).Warn("Error on frame read")
			pc.Close()
			continue
		}

		uAddr := pc.RemoteAddr()

		// Serialize the packet.
		encodedPack := encodeReadTCPPacket(uint16(byteNum), uAddr, payload[:])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
		payload = nil
	}
}

// Opens a TCP connection with the peer sent on the params and transmits
// a stream of bytes. Once transmitted, closes the connection.
func sendTCPStream(addr net.UDPAddr, payload []byte) {
	conn, err := net.Dial("tcp", string(addr.IP)+":"+string(addr.Port))
	if err != nil {
		log.WithError(err).Warn("Could not stablish a connection with the dest Peer.")
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// Write our message to the connection.
	if err = writeTCPFrame(conn, payload); err != nil {
		log.WithError(err).Warnf("Could not write to addr %s", addr.String())
		return
	}
}
