package kadcast

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"

	"golang.org/x/crypto/sha3"

	// Just for debugging purposes
	_ "fmt"
)

const (
	// MaxFrameSize is set based on max block size expected
	MaxFrameSize = 5000000
)

var (
	// ErrExceedMaxLen max message size of a frame on the wire
	ErrExceedMaxLen = errors.New("message size exceeds max frame length")
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
	copy(halfLenID[:], doubleLenID[0:16])
	return halfLenID
}

// ComputePeerID exposes computePeerID
func ComputePeerID(buf [4]byte) [16]byte {
	return computePeerID(buf)
}

// This function is a middleware that allows the peer to verify
// other Peers nonce's and validate them if they are correct.
func verifyIDNonce(id [16]byte, nonce [4]byte) error {
	idPlusNonce := make([]byte, 20)
	copy(idPlusNonce[0:16], id[0:16])
	copy(idPlusNonce[16:20], nonce[0:4])
	hash := sha3.Sum256(idPlusNonce)
	if (hash[31]) == 0 {
		return nil
	}
	return errors.New("Id and Nonce are not valid parameters") //TODO: Create error type.
}

// Returns the ID associated to the chunk sent.
// The ID is the half of the result of the hash of the chunk.
func computeChunkID(chunk []byte) [16]byte {
	var halfLenID [16]byte
	fullID := sha3.Sum256(chunk)
	copy(halfLenID[0:16], fullID[0:16])
	return halfLenID
}

// ------------------ NET UTILS ------------------ //

// Get outbound IP returns local address
// TODO To be replaced with config param
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	_ = conn.Close()

	return localAddr.IP
}

// Format the UDP address, the UDP listener binds on
func getLocalUDPAddress(port int) net.UDPAddr {
	laddr := net.UDPAddr{IP: getOutboundIP()}
	laddr.Port = port
	return laddr
}

// Gets the TCP address, the TCP listener binds on
func getLocalTCPAddress(port int) net.TCPAddr {
	laddr := net.TCPAddr{IP: getOutboundIP()}
	laddr.Port = port
	return laddr
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
		// Cut the input to byte range.
		res[i] = byte(num & 255)
		// Shift it to subtract a byte from the number.
		num = num >> 8
	}
	return res
}

// Encodes received UDP packets to send it through the
// Ring to the packetProcess rutine.
func encodeReadUDPPacket(byteNum uint16, peerAddr net.UDPAddr, payload []byte) []byte {
	encodedLen := len(payload) + 8
	enc := make([]byte, encodedLen)
	// Get numBytes as slice of bytes.
	numBytes := getBytesFromUint16(byteNum)
	// Append it to the resulting slice.
	copy(enc[0:2], numBytes[0:2])
	// Append Peer IP.

	l := len(peerAddr.IP)
	ip := peerAddr.IP[l-4 : l]

	copy(enc[2:6], ip)
	// Append Port
	port := getBytesFromUint16(uint16(peerAddr.Port))
	copy(enc[6:8], port[0:2])
	// Append Payload
	copy(enc[8:encodedLen], payload[0:len(payload)])
	return enc
}

// Encodes received TCP packets to send it through the
// Ring to the packetProcess rutine.
func encodeReadTCPPacket(byteNum uint16, peerAddr net.Addr, payload []byte) []byte {

	encodedLen := len(payload) + 8
	enc := make([]byte, encodedLen)
	// Get numBytes as slice of bytes.
	numBytes := getBytesFromUint16(byteNum)
	// Append it to the resulting slice.
	copy(enc[0:2], numBytes[0:2])
	// Append Peer IP.

	tcpAddr, _ := net.ResolveTCPAddr(peerAddr.Network(), peerAddr.String())
	l := len(tcpAddr.IP)
	ip := tcpAddr.IP[l-4 : l]

	copy(enc[2:6], ip)
	// Append Port
	port := getBytesFromUint16(uint16(tcpAddr.Port))
	copy(enc[6:8], port[0:2])
	// Append Payload
	copy(enc[8:encodedLen], payload[0:len(payload)])
	return enc
}

// Decodes a CircularQueue packet and returns the
// elements of the original received packet.
func decodeRedPacket(packet []byte) (int, *net.UDPAddr, []byte, error) {
	redPackLen := len(packet)
	byteNum := int(binary.LittleEndian.Uint16(packet[0:2]))
	if (redPackLen) != (byteNum + 8) {
		return 0, nil, nil, errors.New("Packet's length taken from the ring differs from expected")
	}
	ip := packet[2:6]
	port := int(binary.LittleEndian.Uint16(packet[6:8]))
	payload := packet[8:]

	peerAddr := net.UDPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}
	return byteNum, &peerAddr, payload, nil
}

func readTCPFrame(r io.Reader) ([]byte, int, error) {

	// Read frame length.
	ln := make([]byte, 4)
	_, err := io.ReadFull(r, ln)
	if err != nil {
		return nil, 0, err
	}

	length := binary.LittleEndian.Uint32(ln[:])
	if length > MaxFrameSize {
		return nil, 0, ErrExceedMaxLen
	}

	// Read packet payload
	var n int
	payload := make([]byte, length)
	if n, err = io.ReadFull(r, payload); err != nil {
		return nil, 0, err
	}

	return payload, n, nil
}

func writeTCPFrame(w io.Writer, payload []byte) error {

	frameLength := uint32(len(payload))
	if frameLength > MaxFrameSize {
		return ErrExceedMaxLen
	}

	// Add packet length
	frame := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(frame, frameLength); err != nil {
		return err
	}

	// Append packet payload
	if _, err := frame.Write(payload); err != nil {
		return err
	}

	// Write data stream
	if _, err := w.Write(frame.Bytes()); err != nil {
		return err
	}

	return nil
}
