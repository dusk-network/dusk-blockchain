package kadcast

import (
	"encoding/binary"
	"sync"
	log "github.com/sirupsen/logrus"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// Packet represents a Kadcast packet which is
// the payload of the TCP or UDP packet received.
type Packet struct {
	headers [24]byte
	payload []byte
}

// Builds a `Packet` from the headers and the payload.
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
	hl := len(pac.headers)
    l := hl + len(pac.payload)
    byteRepr := make([]byte, l)
    copy(byteRepr, pac.headers[:])
	copy(byteRepr[hl:], pac.payload[:])
	return byteRepr
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
func (pac *Packet) setHeadersInfo(tipus byte, router Router, destPeer Peer) {
	headers := make([]byte, 24)
	// Add `Packet` type.
	headers = append(headers[0:1], tipus)
	// Add MyPeer ID
	copy(headers[1:17], router.MyPeerInfo.id[0:16])
	// Attach IdNonce
	idNonce := getBytesFromUint32(router.myPeerNonce)
	copy(headers[17:21], idNonce[0:4])
	// Attach Port
	port := getBytesFromUint16(destPeer.port)
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
	count := getBytesFromUint16(uint16(len(kClosestPeers)))
	pac.payload = append(pac.payload[:], count[:]...)
	// Serialize the Peers to get them in `wire-format`,
	// basically, represented as bytes.
	for _, peer := range kClosestPeers {
		pac.payload = append(pac.payload[:], peer.deserialize()...)
	}
	return len(kClosestPeers)
}

// Analyzes if the announced number of Peers included on the
// `NODES` message payload is the same as the recieved one.
// Returns `true` if it is correct and `false` otherways.
func (pac Packet) checkNodesPayloadConsistency(byteNum int) bool {
	// Get number of Peers announced.
	peerNum := binary.BigEndian.Uint16(pac.payload[0:2])
	// Get peerSlice length subtracting headers and count.
	peerSliceLen := byteNum - (len(pac.headers) + 2)

	return int(peerNum)*PeerBytesSize == peerSliceLen
}

// Gets a `NODES` message and returns a slice of the
// `Peers` found inside of it
func (pac Packet) getNodesPayloadInfo() []Peer {
	// Get number of Peers recieved.
	peerNum := int(binary.BigEndian.Uint16(pac.payload[0:2]))
	// Create Peer-struct slice
	var peers []Peer
	// Slice the payload into `Peers` in bytes format and deserialize
	// every single one of them.
	var i, j int = 3, PeerBytesSize + 1
	for m := 0; m < peerNum; m++ {
		// Get the peer structure from the payload and
		// append the peer to the returned slice of Peer structs.
		peers = append(peers[:], serializePeer(pac.payload[i:j]))

		i += PeerBytesSize
		j += PeerBytesSize
	}
	return peers
}

// ProcessPacket recieves a Packet and processes it according to
// it's type. It gets the packets from the circularqueue that 
// connects the listeners with the packet processor.
func ProcessPacket(queue *ring.Buffer, router *Router, wg *sync.WaitGroup) {
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
		NextItem: for _, item := range queuePackets {
			// Get items from the queue packet taken.
			byteNum, senderAddr, udpPayload, err = decodeRedPacket(item)
			if err != nil {
				log.WithError(err).Warn("Error decoding the packet taken from the ring.")
				break NextItem
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
				log.WithError(err).Warn("Incorrect packet sender ID. Skipping its processing.")
				break NextItem
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
				log.Info("Recieved PING message from %v", peerInf.ip[:])
				handlePing(peerInf, router)
				// For NetwDisc we track the `PING` also since this is what
				// introduces new `Peers` in the Buckets.
				if isBootstrapping || isDiscoveringNetwork {
					wg.Done()
				}
			case 1:
				log.Info("Recieved PONG message from %v", peerInf.ip[:])
				handlePong(peerInf, router)

			case 2:
				log.Info("Recieved FIND_NODES message from %v", peerInf.ip[:])
				handleFindNodes(peerInf, router)

			case 3:
				log.Info("Recieved NODES message from %v", peerInf.ip[:])
				handleNodes(peerInf, packet, router, byteNum)
			}
		}
	}
}

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
		// Since the packet is not consisten, we just discard it.
		log.Info("NODES message recieved with corrupted payload. PeerNum mismatch!\nIgnoring the packet.")
		return
	}

	// Process peer addition to the tree.
	router.tree.addPeer(router.MyPeerInfo, peerInf)

	// Deserialize the payload to get the peerInfo of every
	// recieved peer.
	peers := packet.getNodesPayloadInfo()

	// Send `PING` messages to all of the peers to then
	// add them to our buckets if they respond with a `PONG`.
	for _, peer := range peers {
		router.sendPing(peer)
	}
}
