package kadcast

import (
	"encoding/binary"
	"net"
	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// ProcessTCPPacket recieves a Packet and processes it according to
// it's type. It gets the packets from the circularqueue that
// connects the listeners with the packet processor.
func ProcessTCPPacket(queue *ring.Buffer, router *Router) {
	// Instantiate now the variables to not pollute
	// the stack.
	var err error
	var byteNum int
	var senderAddr *net.UDPAddr
	var tcpPayload []byte
	var packet Packet
	for {
		// Get all of the packets that are now on the queue.
		queuePackets, _ := queue.GetAll()
		for _, item := range queuePackets {
			// Get items from the queue packet taken.
			byteNum, senderAddr, tcpPayload, err = decodeRedPacket(item)
			if err != nil {
				log.WithError(err).Warn("Error decoding the TCP packet taken from the ring.")
				continue
			}
			// Build packet struct
			packet = getPacketFromStream(tcpPayload[:])
			// Extract headers info.
			tipus, senderID, nonce, peerRecepPort := packet.getHeadersInfo()

			// Verify IDNonce
			err = verifyIDNonce(senderID, nonce)
			// If we get an error, we just skip the whole process since the
			// Peer was not validated.
			if err := verifyIDNonce(senderID, nonce); err != nil {
				log.WithError(err).Warn("Incorrect TCP packet sender ID. Skipping its processing.")
				continue
			}

			// Build Peer info and put the right port on it subsituting the one
			// used to send the message by the one where the peer wants to receive
			// the messages.
			ip, _ := getPeerNetworkInfo(*senderAddr)
			port := binary.LittleEndian.Uint16(peerRecepPort[:])
			peerInf := MakePeer(ip, port)

			switch tipus {
			case 0: {
				handleChunks(peerInf, packet, router, byteNum)
				log.WithField(
					"Source-IP", peerInf.ip[:],
				).Infoln("Recieved CHUNKS message")
			}
			}
		}
	}
}

// ProcessUDPPacket recieves a Packet and processes it according to
// it's type. It gets the packets from the circularqueue that
// connects the listeners with the packet processor.
func ProcessUDPPacket(queue *ring.Buffer, router *Router) {
	// Instantiate now the variables to not pollute
	// the stack.
	var err error
	var byteNum int
	var senderAddr *net.UDPAddr
	var udpPayload []byte
	var packet Packet
	for {
		// Get all of the packets that are now on the queue.
		queuePackets, _ := queue.GetAll()
		for _, item := range queuePackets {
			// Get items from the queue packet taken.
			byteNum, senderAddr, udpPayload, err = decodeRedPacket(item)
			if err != nil {
				log.WithError(err).Warn("Error decoding the UDP packet taken from the ring.")
				continue
			}
			// Build packet struct
			packet = getPacketFromStream(udpPayload[:])
			// Extract headers info.
			tipus, senderID, nonce, peerRecepPort := packet.getHeadersInfo()

			// Verify IDNonce
			err = verifyIDNonce(senderID, nonce)
			// If we get an error, we just skip the whole process since the
			// Peer was not validated.
			if err := verifyIDNonce(senderID, nonce); err != nil {
				log.WithError(err).Warn("Incorrect UDP packet sender ID. Skipping its processing.")
				continue
			}

			// Build Peer info and put the right port on it subsituting the one
			// used to send the message by the one where the peer wants to receive
			// the messages.
			ip, _ := getPeerNetworkInfo(*senderAddr)
			port := binary.LittleEndian.Uint16(peerRecepPort[:])
			peerInf := MakePeer(ip, port)

			// Check packet type and process it.
			switch tipus {
			case 0:
				log.WithField(
					"Source-IP", peerInf.ip[:],
				).Infoln("Recieved PING message")
				handlePing(peerInf, router)
			case 1:
				log.WithField(
					"Source-IP", peerInf.ip[:],
				).Infoln("Recieved PONG message")
				handlePong(peerInf, router)

			case 2:
				log.WithField(
					"Source-IP", peerInf.ip[:],
				).Infoln("Recieved FIND_NODES message")
				handleFindNodes(peerInf, router)

			case 3:
				log.WithField(
					"Source-IP", peerInf.ip[:],
				).Infoln("Recieved NODES message")
				handleNodes(peerInf, packet, router, byteNum)
			}
		}
	}
}
