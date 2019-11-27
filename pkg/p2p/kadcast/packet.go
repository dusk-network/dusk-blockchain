package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

type Packet struct {
	headers [24]byte
	payload []byte
}

func makePacket(headers [24]byte, payload []byte) Packet {
	return Packet{
		headers: headers,
		payload: payload,
	}
}

// -------- General Packet De/Serialization tools -------- //

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

// Deserializes the packet into an slice of bytes.
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

// -------- NODES Packet De/Serialization tools -------- //

// Builds the payload of a `NODES` message by collecting,
// deserializing and adding to the packet's payload the
// peerInfo of the `K` closest Peers in respect to a certain
// target Peer.
func (pack *Packet) setNodesPayload(router Router, targetPeer Peer) int {
	// Get `K` closest peers to `targetPeer`.
	kClosestPeers := router.getXClosestPeersTo(K, targetPeer)
	// Compute the ammount of Peers that will be sent and add it
	// as a two-byte array.
	count := getBytesFromUint16(uint16(len(kClosestPeers)))
	pack.payload = append(pack.payload[:], count[:]...)
	// Serialize the Peers to get them in `wire-format`,
	// basically, represented as bytes.
	for _, peer := range kClosestPeers {
		pack.payload = append(pack.payload[:], peer.deserializePeer()...)
	}
	return len(kClosestPeers)
}

// Analyzes if the announced number of Peers included on the
// `NODES` message payload is the same as the recieved one.
// Returns `true` if it is correct and `false` otherways.
func (packet Packet) checkNodesPayloadConsistency(byteNum int) bool {
	// Get number of Peers announced.
	peerNum := binary.BigEndian.Uint16(packet.payload[0:2])
	fmt.Printf("\nPeerNum announced: %v", peerNum)
	// Get peerSlice length subtracting headers and count.
	peerSliceLen := byteNum - 26 

	if int(peerNum)*PeerBytesSize != peerSliceLen {
		return false
	}
	return true
}

// Gets a `NODES` message and returns a slice of the
// `Peers` found inside of it
func (packet Packet) getNodesPayloadInfo() []Peer {
	// Get number of Peers recieved.
	peerNum := int(binary.BigEndian.Uint16(packet.payload[0:2]))
	// Create Peer-struct slice
	var peers []Peer
	// Slice the payload into `Peers` in bytes format and deserialize
	// every single one of them.
	var i, j int = 3, PeerBytesSize + 1
	for m := 0; m < peerNum; m ++{
		// Get the peer structure from the payload and
		// append the peer to the returned slice of Peer structs.
		peers = append(peers[:], serializePeer(packet.payload[i:j]))

		i += PeerBytesSize
		j += PeerBytesSize
		if i >= peerNum-1 {
			break
		}
	}
	return peers
}

// The function recieves a Packet and processes it according to
// it's type.
func processPacket(senderAddr net.UDPAddr, byteNum int, udpPayload []byte, router *Router) {
	// Build packet struct
	packet := getPacketFromStream(udpPayload[:])
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
		log.Printf("Recieved PING message from %v", peerInf.ip[:])
		treatPing(peerInf, router)
	case 1:
		log.Printf("Recieved PONG message from %v", peerInf.ip[:])
		treatPong(peerInf, router)

	case 2:
		log.Printf("Recieved FIND_NODES message from %v", peerInf.ip[:])
		treatFindNodes(peerInf, router)
	case 3:
		log.Printf("Recieved NODES message from %v", peerInf.ip[:])
		treatNodes(peerInf, packet, router, byteNum)
	}
	return
}

func treatPing(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	// Send back a `PONG` message.
	router.sendPong(peerInf)
	return
}

func treatPong(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	return
}

func treatFindNodes(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)
	// Send back a `NODES` message to the peer that
	// send the `FIND_NODES` message.
	router.sendNodes(peerInf)
	return
}

func treatNodes(peerInf Peer, packet Packet, router *Router, byteNum int) {
	// See if the packet info is consistent:
	// peerNum announced <=> bytesPerPeer * peerNum
	if !packet.checkNodesPayloadConsistency(byteNum) {
		// Since the packet is not consisten, we just discard it.
		log.Println("NODES message recieved with corrupted payload. PeerNum mismatch!")
		//return
	}

	// Process peer addition to the tree.
	router.tree.addPeer(router.myPeerInfo, peerInf)

	// Deserialize the payload to get the peerInfo of every
	// recieved peer.
	peers := packet.getNodesPayloadInfo()

	// Send `PING` messages to all of the peers to then
	// add them to our buckets if they respond with a `PONG`.
	for _, peer := range peers {
		router.sendPing(peer)
	}
}
