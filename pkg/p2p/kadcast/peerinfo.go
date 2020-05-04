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

// PeerInfo stores the info of a peer which consists on:
// - IP of the peer.
// - Port to connect to it.
// - The ID of the peer.
type PeerInfo struct {
	ip   [4]byte
	port uint16
	id   [16]byte
}

// MakePeerFromAddr is same as MakePeer but resolve addr with ResolveTCPAddr
func MakePeerFromAddr(addr string) (PeerInfo, error) {

	laddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return PeerInfo{}, err
	}

	l := len(laddr.IP)
	var ip [4]byte
	if l > 4 {
		copy(ip[:], laddr.IP[l-4 : l][:])
	}

	port := uint16(laddr.Port)

	return MakePeer(ip, port), nil
}

// MakePeer builds a peer tuple by computing ID over IP and port
func MakePeer(ip [4]byte, port uint16) PeerInfo {
	id := computePeerID(ip, port)
	return PeerInfo{ip, port, id}
}

// TODO: Refactoring packet and peer marshaling
// Deserializes a `Peer` structure as an array of bytes
// that allows to send it through a wire.
func marshalPeerInfo(peer *PeerInfo) []byte {
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
func unmarshalPeerInfo(peerBytes []byte) PeerInfo {
	// Get Ip
	var ip [4]byte
	copy(ip[:], peerBytes[0:4])

	// Get Port
	port := binary.LittleEndian.Uint16(peerBytes[4:6])
	// Get Id
	var id [16]byte
	copy(id[:], peerBytes[6:22])

	return PeerInfo{
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
func (peer PeerInfo) computePeerNonce() uint32 {
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
func (peer PeerInfo) computeDistance(otherPeer PeerInfo) (uint16, [16]byte) {
	return idXor(peer.id, otherPeer.id)
}

// Reads the network info of a `Peer` and returns its
// corresponding `UDPAddr` struct.
func (peer PeerInfo) getUDPAddr() net.UDPAddr {
	return net.UDPAddr{
		IP:   peer.ip[:],
		Port: int(peer.port),
		Zone: "",
	}
}

// IsEqual returns true if two peers tuples are identical
func (peer PeerInfo) IsEqual(p PeerInfo) bool {

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

func (peer PeerInfo) String() string {

	id := hex.EncodeToString(peer.id[:])
	if len(id) >= 7 {
		id = id[0:7]
	}

	return fmt.Sprintf("%s, %s", peer.Address(), id)
}

// Address returns peer address as a string
func (peer PeerInfo) Address() string {

	if len(peer.ip) < 4 {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.%d:%d", peer.ip[0], peer.ip[1], peer.ip[2], peer.ip[3], peer.port)
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
