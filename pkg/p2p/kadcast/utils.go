package kadcast

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math/big"
	"math/bits"
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
	//ErrExceedMaxLen is the error thrown if the message size exceeds the max
	//frame length
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
func idXor(a [16]byte, b [16]byte) (uint16, [16]byte) {
	distance := xor(a, b)
	return classifyDistance(distance), distance
}

// classifyDistance calculates floor of log2 of the distance between two nodes
// As per that, classifyDistance returns rank of most significant bit in LE format
func classifyDistance(distance [16]byte) uint16 {

	for i := len(distance) - 1; i >= 0; i-- {
		if distance[i] == 0 {
			continue
		}
		// Len8 returns the minimum number of bits required to represent x
		// That said, most significant bit rank in Little Endian
		msbRank := bits.Len8(distance[i])
		msbRank--

		pos := uint16(msbRank + i*8)
		return pos
	}
	return 0
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
func computePeerID(ip [4]byte, port uint16) [16]byte {

	seed := make([]byte, 2)
	binary.LittleEndian.PutUint16(seed, port)
	seed = append(seed, ip[:]...)

	doubleLenID := sha3.Sum256(seed[:])
	var halfLenID [16]byte
	copy(halfLenID[:], doubleLenID[0:16])

	return halfLenID
}

/*
// computePeerDummyID is helpful on simplifying ID on local net
func computePeerDummyID(ip [4]byte, port uint16) [16]byte {
	var id [16]byte
	port -= 10000
	seed := make([]byte, 16)
	binary.LittleEndian.PutUint16(seed, port)
	copy(id[:], seed[0:16])
	return id
}
*/

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
	defer func() {
		_ = conn.Close()
	}()

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
	copy(enc[8:encodedLen], payload[0:])
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
	copy(enc[8:encodedLen], payload[0:])
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

// generateRandomDelegates selects n random and distinct items from `in` and
// copy them into `out` slice (no duplicates)
func generateRandomDelegates(beta uint8, in []Peer, out *[]Peer) error {

	if in == nil || out == nil {
		return errors.New("invalid in/out params")
	}

	if len(in) == 0 || len(*out) == int(beta) {
		return nil
	}

	maxval := int64(len(in))
	// #654
	nBig, err := rand.Int(rand.Reader, big.NewInt(maxval))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	ind := uint32(n)

	*out = append(*out, in[ind])

	in[ind] = in[len(in)-1]
	in = in[:len(in)-1]

	return generateRandomDelegates(beta, in, out)
}
