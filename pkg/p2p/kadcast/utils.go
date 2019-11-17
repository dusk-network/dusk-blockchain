package kadcast

import (
	"log"
	"errors"
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

// Performs the hash of the wallet public
// IP address and gets the first 16 bytes of 
// it.
func computePeerID(externIP [4]byte) [16]byte {
	var halfLenID [16]byte
	doubleLenID := sha3.Sum256(externIP[:])
	copy(halfLenID[:], doubleLenID[0:15])
	return halfLenID
}

// This function is a middleware that allows the peer to verify
// other Peers nonce's and validate them if they are correct.
func verifyIDNonce(id [16]byte, senderAddr net.UDPAddr, nonce [4]byte) (*Peer, error) {
	hash := sha3.Sum256(append(id[:], nonce[:]...))
	if (hash[31] | hash[30] | hash[29]) == 0 {
		var peerIP [4]byte
		copy(peerIP[:], senderAddr.IP[:])

		return &Peer {
			ip: peerIP,
			port: uint16(senderAddr.Port),
			id: id,
		}, nil
	}
	return nil, errors.New("\nId and Nonce are not valid parameters.") //Create error type.
}

// ------------------ NET UTILS ------------------ //

// Gets the local IP address of the machine where
// the node is running.
//
// Panics if it there's not connection.
func getLocalIPAddress() net.UDPAddr {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return *localAddr
}

// ------------------ ENC/DEC UTILS ------------------ //
// Set a `uint32` in bytes format.
func getBytesFromUint(num uint32) *[4]byte {
	res := [4]byte{0, 0, 0, 0}
	for i := 0; num > 0; i++ {
		res[i] = byte(num & 255)
		num = num >> 8
	}
	return &res
}
// Get an `uint32` from a 4-byte array.
func getUintFromBytes(arr *[4]byte) *uint32 {
	var res uint32 = 0
	for i := 0; i < 4; i++ {
		res += uint32(uint32(arr[i]) << uint32(8*i))
	}
	return &res
}