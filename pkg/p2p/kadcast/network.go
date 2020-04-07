package kadcast

import (
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// StartUDPListener listens infinitely for UDP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func StartUDPListener(netw string, queue *ring.Buffer, MyPeerInfo Peer) {

	lAddr := getLocalUDPAddress(int(MyPeerInfo.port))
	// listen to incoming udp packets
	listener, err := net.ListenUDP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		_ = listener.Close()
	}()

	log.Infof("Starting UDP Listener on %s", lAddr.String())

	for {
		// Read UDP packet.
		_ = listener.SetDeadline(time.Now().Add(5 * time.Minute))
		buffer := make([]byte, 1024)
		byteNum, uAddr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			return
		}
		// Serialize the packet.
		encodedPack := encodeReadUDPPacket(uint16(byteNum), *uAddr, buffer[0:byteNum])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
		buffer = nil
	}
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// receiver `Peer` and sends to it a UDP Packet with the payload inside.
func sendUDPPacket(laddr, raddr net.UDPAddr, payload []byte) {

	log.WithField("dest", raddr.String()).Tracef("Dialing udp")

	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0
	conn, err := net.DialUDP("udp", &laddr, &raddr)
	if err != nil {
		log.WithError(err).Warn("Could not establish a connection with the dest Peer.")
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	log.WithField("src", conn.LocalAddr().String()).
		WithField("dest", raddr.String()).Traceln("Sending udp")

	// Simple write
	_, err = conn.Write(payload)
	if err != nil {
		log.WithError(err).Warn("Error while writing to the filedescriptor.")
		return
	}
}

// StartTCPListener listens infinitely for TCP packet arrivals and
// executes it's processing inside a gorutine by sending
// the packets to the circularQueue.
func StartTCPListener(netw string, queue *ring.Buffer, MyPeerInfo Peer) {

	lAddr := getLocalTCPAddress(int(MyPeerInfo.port))
	// listen to incoming TCP packets
	listener, err := net.ListenTCP(netw, &lAddr)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		_ = listener.Close()
	}()

	log.Infof("Starting TCP Listener on %s", lAddr.String())

	for {
		_ = listener.SetDeadline(time.Now().Add(5 * time.Minute))
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Warn("Error on tcp accept")
			return
		}

		// Read frame payload
		// Set a new deadline for the connection.
		_ = conn.SetDeadline(time.Now().Add(time.Minute))

		payload, byteNum, err := readTCPFrame(conn)
		if err != nil {
			log.WithError(err).Warn("Error on frame read")
			_ = conn.Close()
			continue
		}

		// Serialize the packet.
		encodedPack := encodeReadTCPPacket(uint16(byteNum), conn.RemoteAddr(), payload[:])
		// Send the packet to the Consumer putting it on the queue.
		queue.Put(encodedPack)
		payload = nil

		// Current impl expects only one TCPFrame per connection
		_ = conn.Close()
	}
}

// Opens a TCP connection with the peer sent on the params and transmits
// a stream of bytes. Once transmitted, closes the connection.
func sendTCPStream(raddr net.UDPAddr, payload []byte) {

	address := raddr.IP.String() + ":" + strconv.Itoa(raddr.Port)
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		log.WithError(err).Warnf("Could not establish a peer connection %s.", raddr.String())
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	log.WithField("src", conn.LocalAddr().String()).
		WithField("dest", raddr.String()).Traceln("Sending tcp")

	// Write our message to the connection.
	if err = writeTCPFrame(conn, payload); err != nil {
		log.WithError(err).Warnf("Could not write to addr %s", raddr.String())
		return
	}
}
