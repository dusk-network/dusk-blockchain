package kadcast

import (
	"encoding/binary"
	"errors"
	"logrus"
	"net"

	"golang.org/x/crypto/sha3"

	// Just for debugging purposes
	_ "fmt"
)

// ------------------ DISTANCE UTILS ------------------ //

// Computes the XOR between two [16]byte arrays.
func xor(a [16]byte, b [16]byte) [16]byte {
	distance := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	for i := 0; i < 16; i++ {
		distance[i] = a[i] ^ b[i]
	}
	return distance
}

// Computes the XOR distance between 2 different
// ids and classifies it between the range 0-128.
func idXor(a [16]byte, b [16]byte) uint16 {
	distance := xor(a, b)
	return classifyDistance(distance)
}

// This function gets the XOR distance as a byte-array
// and collapses it to classify the distance on one of the
// 128 buckets.
func classifyDistance(arr [16]byte) uint16 {
	var collDist uint16 = 0
	for i := 0; i < 16; i++ {
		collDist += countSetBits(arr[i])
	}
	return collDist
}

// Counts the number of setted bits in the given byte.
func countSetBits(byt byte) uint16 {
	var count uint16 = 0
	for byt != 0 {
		count += uint16(byt & 1)
		byt >>= 1
	}
	return count
}

// Evaluates if an XOR-distance of two peers is
// bigger than another.
func xorIsBigger(a [16]byte, b [16]byte) bool {
	for i := 15; i > 0; i-- {
		if a[i] < b[i] {
			return false
		}
	}
	return true
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
func verifyIDNonce(id [16]byte, nonce [4]byte) error {
	idPlusNonce := make([]byte, 20)
	copy(idPlusNonce[0:16], id[0:16])
	copy(idPlusNonce[16:20], nonce[0:4])
	hash := sha3.Sum256(idPlusNonce)
	if (hash[31] | hash[30] | hash[29]) == 0 {
		return nil
	}
	return errors.New("\nId and Nonce are not valid parameters.") //TODO: Create error type.
}

// ------------------ NET UTILS ------------------ //

// Gets the local IP address of the machine where
// the node is running in `net.UDPAddr` format.
//
// Panics if it there's not connection.
func getLocalUDPAddress() net.UDPAddr {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logrus.WithError(err).Warn("Network Unreachable.")
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return *localAddr
}

// Gets the local IP address of the machine where
// the node is running in `net.UDPAddr` format.
//
// Panics if it there's not connection.
func getLocalTCPAddress() net.TCPAddr {
	conn, err := net.Dial("tcp", "8.8.8.8:80")
	if err != nil {
		logrus.WithError(err).Warn("Network Unreachable.")
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	return *localAddr
}


// ------------------ ENC/DEC UTILS ------------------ //

// Set a `uint32` in bytes format.
func getBytesFromUint32(num uint32) [4]byte {
	res := [4]byte{0, 0, 0, 0}
	for i := 0; num > 0; i++ {
		res[i] = byte(num & 255)
		num = num >> 8
	}
	return res
}

// Set a `uint16` in bytes format.
func getBytesFromUint16(num uint16) [2]byte {
	res := [2]byte{0, 0}
	for i := 0; num > 0; i++ {
		// Cut the input to byte rabge.
		res[i] = byte(num & 255)
		// Shift it to subtract a byte from the number.
		num = num >> 8
	}
	return res
}

// Encode a received packet to send it through the
// Ring to the packetProcess rutine.
func encodeRedPacket(byteNum uint16, peerAddr net.UDPAddr, payload []byte) []byte {
	encodedLen := len(payload) + 8
	enc := make([]byte, encodedLen)
	// Get numBytes as slice of bytes.
	numBytes := getBytesFromUint16(byteNum)
	// Append it to the resulting slice.
	copy(enc[0:2], numBytes[0:2])
	// Append Peer IP.
	copy(enc[2:6], peerAddr.IP[0:4])
	// Append Port
	port := getBytesFromUint16(uint16(peerAddr.Port))
	copy(enc[6:8], port[0:2])
	// Append Payload
	copy(enc[8:encodedLen], payload[0:len(payload)])
	return enc
}

// Decode a CircularQueue packet and return the
// elements of the original received packet.
func decodeRedPacket(packet []byte) (int,  *net.UDPAddr, []byte, error) {
	redPackLen := len(packet)
	byteNum := int(binary.LittleEndian.Uint16(packet[0:2]))
	if (redPackLen) != (byteNum + 8) {
		return 0, nil, nil, errors.New("\nPacket's length taken from the ring differs from expected.")
	}
	ip := packet[2:6]
	port := int(binary.LittleEndian.Uint16(packet[6:8]))
	payload := packet[8:]
	
	peerAddr := net.UDPAddr {
		IP: ip,
		Port: port,
		Zone: "N/A",
	}
	return byteNum, &peerAddr, payload, nil
}
