package kadcast

import (
	"log"
	"net"
)

type Packet struct {
	headers [22]byte
	payload []byte
}

// Gets a stream of bytes and slices it between headers of Kadcast
// protocol and the payload.
func getPacketFromStream(stream []byte) Packet {
	var headers [22]byte
	copy(headers[:], stream[0:21])
	return Packet{
		headers: headers,
		payload: stream[22:],
	}
}

// Returns the headers info sliced into three pieces:
// Packet type, SenderId and IdNonce.
func (pac Packet) getHeadersInfo() (byte, [16]byte, [4]byte) {
	// Construct type, senderID and Nonce
	typ := pac.headers[0]
	var senderID [16]byte
	copy(senderID[:], pac.headers[1:17])
	var nonce [4]byte
	copy(nonce[:], pac.headers[18:21])
	return typ, senderID, nonce
}

// The function recieves a Packet and processes it according to 
// it's type.
func processPacket(senderAddr net.UDPAddr, byteNum int, payload []byte, router *Router) {
	log.Printf("Revieved new packet from: %s", senderAddr.IP)
	// Build packet struct
	packet := getPacketFromStream(payload[:])
	// Extract headers info.
	tipus, senderID, nonce := packet.getHeadersInfo()

	// Verify IDNonce
	peerInf, err := verifyIDNonce(senderID, senderAddr, nonce)
	// If we get an error, we just skip the whole process since the
	// Peer was not validated.
	if err != nil {
		log.Printf("%s", err)
		return
	}
	// Check packet type and process it.	
	switch tipus {
	case 0:
		treatPing(*peerInf, router)
		log.Printf("Recieved PING message from %v", peerInf.ip[:])
	case 1:
		treatPong(*peerInf, router)
		log.Printf("Recieved PONG message from %v", peerInf.ip[:])
	}
}

func treatPing(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	// Send back a `Pong` message.
	router.sendPong(peerInf)
}

func treatPong(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
}
