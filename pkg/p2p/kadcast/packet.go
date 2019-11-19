package kadcast

import (
	"encoding/binary"
	"log"
	"net"
)

type Packet struct {
	headers [24]byte
	payload []byte
}

func makePacket(headers [24]byte, payload []byte) Packet {
	return Packet {
		headers: headers,
		payload: payload,
	}
}

// Gets a stream of bytes and slices it between headers of Kadcast
// protocol and the payload.
func getPacketFromStream(stream []byte) Packet {
	var headers [24]byte
	copy(headers[:], stream[0:23])
	return Packet{
		headers: headers,
		payload: stream[23:],
	}
}

func (pac Packet) asBytes() []byte {
	return append(pac.headers[:], pac.payload[:]...)
}

// Returns the headers info sliced into three pieces:
// Packet type, SenderId, IdNonce and senderPort.
func (pac Packet) getHeadersInfo() (byte, [16]byte, [4]byte, [2]byte) {
	// Construct type, senderID and Nonce
	typ := pac.headers[0]
	var senderID [16]byte
	copy(senderID[:], pac.headers[1:17])
	var nonce [4]byte
	copy(nonce[:], pac.headers[17:21])
	var peerPort [2]byte
	copy(peerPort[:], pac.headers[21:23])
	return typ, senderID, nonce, peerPort
}

// Gets the Packet headers parts and puts them into the
// header attribute of the Packet.
func (pack *Packet) setHeadersInfo(tipus byte, router Router, destPeer Peer) {
	var headers []byte
	// Add `Packet` type.
	headers = append(headers[:], tipus)
	// Add MyPeer ID
	headers = append(headers[:], router.myPeerInfo.id[:]...)
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.myPeerNonce)
	headers = append(headers[:], idNonce[:]...)
	// Attach Port
	port := getBytesFromUint16(destPeer.port)
	headers = append(headers[:], port[:]...)

	// Build headers array from the slice.
	var headersArr [24]byte
	copy(headersArr[:], headers[0:24])

	pack.headers = headersArr
}


// The function recieves a Packet and processes it according to
// it's type.
func processPacket(senderAddr net.UDPAddr, byteNum int, payload []byte, router *Router) {
	// Build packet struct
	packet := getPacketFromStream(payload[:])
	// Extract headers info.
	tipus, senderID, nonce, peerRecepPort := packet.getHeadersInfo()

	// Verify IDNonce
	err := verifyIDNonce(senderID, nonce)
	// If we get an error, we just skip the whole process since the
	// Peer was not validated.
	if err != nil {
		log.Printf("%s", err)
		return
	}

	// Build Peer info and put the right port on it subsituting the one
	// used to send the message by the one where the peer wants to receive
	// the messages.
	ip, _ := getPeerNetworkInfo(senderAddr)
	port := binary.LittleEndian.Uint16(peerRecepPort[:])
	peerInf := makePeer(ip, port)

	// Check packet type and process it.
	switch tipus {
	case 0:
		treatPing(peerInf, router)
		log.Printf("Recieved PING message from %v", peerInf.ip[:])
	case 1:
		treatPong(peerInf, router)
		log.Printf("Recieved PONG message from %v", peerInf.ip[:])
	}
	return
}

func treatPing(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	// Send back a `Pong` message.
	router.sendPong(peerInf)
	return
}

func treatPong(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	return
}
