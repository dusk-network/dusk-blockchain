package kadcast

import (
	"encoding/binary"
	"errors"
)

const (

	// MaxTCPacketSize is the max size allowed of TCP packet
	MaxTCPacketSize = 500000

	// MaxUDPacketSize max size of a UDP packet. As per default MTU 1500, UDP
	// packet should be up to 1472 bytes
	MaxUDPacketSize = 1472
)

// Packet represents a Kadcast packet which is
// the payload of the TCP or UDP packet received.
type Packet struct {
	headers [24]byte
	payload []byte
}

// -------- General Packet De/Serialization tools -------- //

func unmarshalPacket(buf []byte, p *Packet, maxSize int) (uint16, byte, error) {

	headerLen := len(p.headers)
	if len(buf) < headerLen {
		return 0, 0, errors.New("empty packet header")
	}

	copy(p.headers[:], buf[0:headerLen])

	if len(buf) > maxSize {
		return 0, 0, errors.New("packet size exceeds max length")
	}

	p.payload = buf[headerLen:]

	// Extract headers info.
	msgType, senderID, nonce, srcPort := p.getHeadersInfo()

	// Verify IDNonce If we get an error, we just skip the whole process since
	// the Peer was not validated.
	if err := verifyIDNonce(senderID, nonce); err != nil {
		log.WithError(err).Warn("Invalid sender ID")
		return 0, msgType, err
	}

	port := binary.LittleEndian.Uint16(srcPort[:])
	return port, msgType, nil
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
func (pac *Packet) setHeadersInfo(tipus byte, router *RoutingTable) {
	headers := make([]byte, 24)
	// Add `Packet` type.
	headers[0] = tipus
	// Add MyPeer ID
	copy(headers[1:17], router.LpeerInfo.id[0:16])
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.localPeerNonce)
	copy(headers[17:21], idNonce[0:4])
	// Attach Port
	port := getBytesFromUint16(router.LpeerInfo.port)
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
func (pac *Packet) setNodesPayload(router *RoutingTable, targetPeer PeerInfo) int {
	// Get `K` closest peers to `targetPeer`.
	kClosestPeers := router.getXClosestPeersTo(DefaultKNumber, targetPeer)
	// Compute the amount of Peers that will be sent and add it
	// as a two-byte array.
	numBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(numBytes, uint16(len(kClosestPeers)))
	pac.payload = append(pac.payload[:], numBytes...)
	// Serialize the Peers to get them in `wire-format`,
	// basically, represented as bytes.
	for _, peer := range kClosestPeers {
		pac.payload = append(pac.payload[:], marshalPeerInfo(&peer)...)
	}
	return len(kClosestPeers)
}

// Analyzes if the announced number of Peers included on the
// `NODES` message payload is the same as the received one.
// Returns `true` if it is correct and `false` otherwise.
func (pac Packet) checkNodesPayloadConsistency(byteNum int) bool {

	// Get number of Peers announced.
	peerNum := binary.LittleEndian.Uint16(pac.payload[0:2])
	// Get peerSlice length subtracting headers and count.
	peerSliceLen := byteNum - (len(pac.headers) + 2)

	return int(peerNum)*PeerBytesSize == peerSliceLen
}

// Gets a `NODES` message and returns a slice of the
// `Peers` found inside of it
func (pac Packet) getNodesPayloadInfo() []PeerInfo {
	// Get number of Peers received.
	peerNum := int(binary.LittleEndian.Uint16(pac.payload[0:2]))
	// Create Peer-struct slice
	var peers []PeerInfo
	// Slice the payload into `Peers` in bytes format and deserialize
	// every single one of them.
	var i int = 2
	j := i + PeerBytesSize
	for m := 0; m < peerNum; m++ {
		// Get the peer structure from the payload and
		// append the peer to the returned slice of Peer structs.
		p := unmarshalPeerInfo(pac.payload[i:j])
		peers = append(peers[:], p)

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
