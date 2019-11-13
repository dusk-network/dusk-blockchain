package kadcast

import (
	"log"
	"net"

	"golang.org/x/crypto/sha3"

	// Just for debugging purposes
	_ "fmt"
)

// ------------------ DISTANCE UTILS ------------------ //
// Computes the XOR distance between 2 different
// ids.
func idXor(a *[16]byte, b *[16]byte) uint16 {
	distance := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	i := 0

	for i < 16 {
		distance[i] = a[i] ^ b[i]
		i++
	}
	return classifyDistance(&distance)
}

// This function gets the XOR distance as a byte-array
// and collapses it to classify the distance on one of the
// 128 buckets.
func classifyDistance(arr *[16]byte) uint16 {
	var collDist uint16 = 0
	for i := 0; i < 16; i++ {
		collDist += countSetBits(&arr[i])
	}
	return collDist
}

// Counts the number of setted bits in the given byte.
func countSetBits(byt *byte) uint16 {
	var count uint16 = 0
	for *byt != 0 {
		count += uint16(*byt & 1)
		*byt >>= 1
	}
	return count
}

// ------------------ HASH KEY UTILS ------------------ //

// Performs the hash of the wallet Sk
// and uses it as the ID of a Peer.
func computePeerID(key *[32]byte) [16]byte {
	var halfLenID [16]byte
	doubleLenID := sha3.Sum256(key[:])
	copy(halfLenID[:], doubleLenID[0:15])
	return halfLenID
}

// ------------------ NET UTILS ------------------ //

// Reads the network info of a `Peer` and returns the
// corresponding `UDPAddr` struct.
func (peer *Peer) getUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   peer.ip[:],
		Port: int(peer.port),
		Zone: "N/A",
	}
}

// Builds the Peer info from a UPD packet data.
func getPeerNetworkInfo(udpAddress *net.UDPAddr) ([4]byte, uint8) {
	var ip [4]byte
	copy(ip[:], udpAddress.IP[:])
	return ip, uint8(udpAddress.Port)
}

// Gets the local IP address of the machine where
// the node is running.
//
// Panics if it there's not connection.
func getLocalIPAddress() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
