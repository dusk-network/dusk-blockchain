package kadcast

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"

	"golang.org/x/crypto/sha3"
)

// PeerBytesSize represents the amount of bytes
// necessary to represent .
const PeerBytesSize int = 22

// Peer stores the info of a peer which consists on:
// - IP of the peer.
// - Port to connect to it.
// - The ID of the peer.
type Peer struct {
	ip   [4]byte
	port uint16
	id   [16]byte
}

// MakePeer constructs a `Peer` by setting it's IP, Port
// and computing and setting it's ID.
func MakePeer(ip [4]byte, port uint16) Peer {
	id := computePeerID(ip, port)
	peer := Peer{ip, port, id}
	return peer
}

// Deserializes a `Peer` structure as an array of bytes
// that allows to send it through a wire.
func marshalPeer(peer *Peer) []byte {
	serPeer := make([]byte, 22)
	// Add Peer IP.

	copy(serPeer[0:4], peer.ip[0:4])
	// Serialize and add Peer port.
	portByt := make([]byte, 2)
	binary.LittleEndian.PutUint16(portByt, peer.port)
	copy(serPeer[4:6], portByt[0:2])
	// Add Peer ID.
	copy(serPeer[6:22], peer.id[0:16])
	return serPeer
}

// Serializes an array of bytes that contains a Peer
// on it returning a `Peer` structure.
func unmarshalPeer(peerBytes []byte) Peer {
	// Get Ip
	var ip [4]byte
	copy(ip[:], peerBytes[0:4])

	// Get Port
	port := binary.LittleEndian.Uint16(peerBytes[4:6])
	// Get Id
	var id [16]byte
	copy(id[:], peerBytes[6:22])

	return Peer{
		ip:   ip,
		port: port,
		id:   id,
	}
}

// The function receives the user's `Peer` and computes the
// ID nonce in order to be able to join the network.
//
// This operation is basically a PoW algorithm that ensures
// that Sybil attacks are more costly.
func (peer Peer) computePeerNonce() uint32 {
	var nonce uint32 = 0
	var hash [32]byte
	data := make([]byte, 20)
	id := peer.id
	for {
		bytesUint := getBytesFromUint32(nonce)
		copy(data[0:16], id[0:16])
		copy(data[16:20], bytesUint[0:4])
		hash = sha3.Sum256(data)
		if (hash[31]) == 0 {
			return nonce
		}
		nonce++
	}
}

// computeDistance returns both bucket number and XOR distance between two Peers.
func (peer Peer) computeDistance(otherPeer Peer) (uint16, [16]byte) {
	return idXor(peer.id, otherPeer.id)
}

// Reads the network info of a `Peer` and returns its
// corresponding `UDPAddr` struct.
func (peer Peer) getUDPAddr() net.UDPAddr {
	return net.UDPAddr{
		IP:   peer.ip[:],
		Port: int(peer.port),
		Zone: "",
	}
}

func (peer Peer) String() string {
	return fmt.Sprintf("%v:%d, %s", peer.ip, peer.port, hex.EncodeToString(peer.id[:])[0:6])
}

// IsEqual returns true if two peers tuples are identical
func (peer Peer) IsEqual(p Peer) bool {

	if !bytes.Equal(peer.id[:], p.id[:]) {
		return false
	}

	if !bytes.Equal(peer.ip[:], p.ip[:]) {
		return false
	}

	if peer.port != p.port {
		return false
	}

	return true
}

// Builds the Peer info from a UPDAddress struct.
func getPeerNetworkInfo(udpAddress net.UDPAddr) [4]byte {
	var ip [4]byte
	copy(ip[:], udpAddress.IP[:])
	return ip
}

// PeerSort is a helper type to sort `Peers`
type PeerSort struct {
	ip        [4]byte
	port      uint16
	id        [16]byte
	xorMyPeer [16]byte
}
