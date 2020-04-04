package kadcast

import (
	"encoding/binary"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Packet represents a Kadcast packet which is
// the payload of the TCP or UDP packet received.
type Packet struct {
	headers [24]byte
	payload []byte
}

// -------- General Packet De/Serialization tools -------- //

// Gets a stream of bytes and slices it between headers of Kadcast
// protocol and the payload.
func unmarshalPacket(buf []byte, p *Packet) {
	copy(p.headers[:], buf[0:24])
	p.payload = buf[24:]
}

// Deserializes the packet into an slice of bytes.
func marshalPacket(p Packet) []byte {
	b := make([]byte, 0)
	b = append(b, p.headers[:]...)
	b = append(b, p.payload[:]...)
	return b
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

// Gets the Packet headers items and puts them into the
// header attribute of the Packet.
func (pac *Packet) setHeadersInfo(tipus byte, router Router) {
	headers := make([]byte, 24)
	// Add `Packet` type.
	headers[0] = tipus
	// Add MyPeer ID
	copy(headers[1:17], router.MyPeerInfo.id[0:16])
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.myPeerNonce)
	copy(headers[17:21], idNonce[0:4])
	// Attach Port
	port := getBytesFromUint16(router.MyPeerInfo.port)
	copy(headers[21:23], port[0:2])

	// Build headers array from the slice.
	var headersArr [24]byte
	copy(headersArr[:], headers[0:23])

	pac.headers = headersArr
}

// -------- NODES Packet De/Serialization tools -------- //

// Builds the payload of a `NODES` message by collecting,
// deserializing and adding to the packet's payload the
// peerInfo of the `K` closest Peers in respect to a certain
// target Peer.
func (pac *Packet) setNodesPayload(router Router, targetPeer Peer) int {
	// Get `K` closest peers to `targetPeer`.
	kClosestPeers := router.getXClosestPeersTo(K, targetPeer)
	// Compute the amount of Peers that will be sent and add it
	// as a two-byte array.
	numBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(numBytes, uint16(len(kClosestPeers)))
	pac.payload = append(pac.payload[:], numBytes...)
	// Serialize the Peers to get them in `wire-format`,
	// basically, represented as bytes.
	for _, peer := range kClosestPeers {
		pac.payload = append(pac.payload[:], peer.deserialize()...)
	}
	return len(kClosestPeers)
}

// Analyzes if the announced number of Peers included on the
// `NODES` message payload is the same as the received one.
// Returns `true` if it is correct and `false` otherways.
func (pac Packet) checkNodesPayloadConsistency(byteNum int) bool {

	// Get number of Peers announced.
	peerNum := binary.LittleEndian.Uint16(pac.payload[0:2])
	// Get peerSlice length subtracting headers and count.
	peerSliceLen := byteNum - (len(pac.headers) + 2)

	return int(peerNum)*PeerBytesSize == peerSliceLen
}

// Gets a `NODES` message and returns a slice of the
// `Peers` found inside of it
func (pac Packet) getNodesPayloadInfo() []Peer {
	// Get number of Peers received.
	peerNum := int(binary.LittleEndian.Uint16(pac.payload[0:2]))
	// Create Peer-struct slice
	var peers []Peer
	// Slice the payload into `Peers` in bytes format and deserialize
	// every single one of them.
	var i int = 2
	j := i + PeerBytesSize
	for m := 0; m < peerNum; m++ {
		// Get the peer structure from the payload and
		// append the peer to the returned slice of Peer structs.
		peers = append(peers[:], serializePeer(pac.payload[i:j]))

		i += PeerBytesSize
		j += PeerBytesSize
	}
	return peers
}

// -------- CHUNKS Packet De/Serialization tools -------- //

// Sets the payload of a `CHUNKS` message by setting the
// initial height, chunk ID and finally the payload.
func (pac *Packet) setChunksPayloadInfo(height byte, payload []byte) {
	payloadLen := len(payload)
	packPayload := make([]byte, payloadLen+17)
	// Set packet height.
	packPayload[0] = height
	// Set packet ChunkID.
	chunkID := computeChunkID(payload)
	copy(packPayload[1:17], chunkID[0:16])
	// Set the rest of the payload.
	copy(packPayload[17:], payload[0:payloadLen])
	pac.payload = packPayload
}

// Gets the payload of a `CHUNKS` message and deserializes
// it returning height, chunkID and the payload.
func (pac Packet) getChunksPayloadInfo() (byte, *[16]byte, []byte, error) {
	// Get payload length.
	payloadLen := len(pac.payload)
	if payloadLen < 17 {
		return 0, nil, nil, errors.New("payload length insuficient")
	}
	height := pac.payload[0]
	var chunkID [16]byte
	copy(chunkID[0:16], pac.payload[1:17])
	payload := pac.payload[17:]
	return height, &chunkID, payload, nil
}

// ----------- Message Handlers ----------- //

// Processes the `PING` packet info sending back a
// `PONG` message and adding the sender to the buckets.
func handlePing(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.MyPeerInfo, peerInf)
	// Send back a `PONG` message.
	router.sendPong(peerInf)
}

// Processes the `PONG` packet info and
// adds the sender to the buckets.
func handlePong(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.MyPeerInfo, peerInf)
}

// Processes the `FIND_NODES` packet info sending back a
// `NODES` message and adding the sender to the buckets.
func handleFindNodes(peerInf Peer, router *Router) {
	// Process peer addition to the tree.
	router.tree.addPeer(router.MyPeerInfo, peerInf)
	// Send back a `NODES` message to the peer that
	// send the `FIND_NODES` message.
	router.sendNodes(peerInf)
}

// Processes the `NODES` packet info sending back a
// `PING` message to all of the Peers announced on the packet
// and adding the sender to the buckets.
func handleNodes(peerInf Peer, packet Packet, router *Router, byteNum int) {
	// See if the packet info is consistent:
	// peerNum announced <=> bytesPerPeer * peerNum
	if !packet.checkNodesPayloadConsistency(byteNum) {
		// Since the packet is not consistent, we just discard it.
		log.Info("NODES message received with corrupted payload. Packet ignored.")
		return
	}

	// Process peer addition to the tree.
	router.tree.addPeer(router.MyPeerInfo, peerInf)

	// Deserialize the payload to get the peerInfo of every
	// received peer.
	peers := packet.getNodesPayloadInfo()

	// Send `PING` messages to all of the peers to then
	// add them to our buckets if they respond with a `PONG`.
	for _, peer := range peers {
		router.sendPing(peer)
	}
}

func handleChunks(packet Packet, router *Router) error {
	// Deserialize the packet.
	height, chunkID, payload, err := packet.getChunksPayloadInfo()
	if err != nil {
		log.Info("Empty CHUNKS payload. Packet ignored.")
		return err
	}
	// Verify chunkID on the memmoryMap. If we already have it stored,+
	// means that the packet is repeated and we just ignore it.
	if _, ok := router.ChunkIDmap[*chunkID]; ok {
		return fmt.Errorf("chunk ID already registered: %v", *chunkID)
	}

	// Set chunkIDmap to true on the map.
	router.ChunkIDmap[*chunkID] = nil
	if router.StoreChunks {
		router.ChunkIDmap[*chunkID] = payload
	}

	// Verify height, if != 0, decrease it by one and broadcast the
	// packet again.
	if height > 0 {
		router.broadcastPacket(height-1, 0, payload)
	}

	// TODO: HERE WE SHOULD SEND THE PAYLOAD TO THE `EVENTBUS`.
	return nil
}
