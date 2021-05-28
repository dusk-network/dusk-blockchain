// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"math/bits"
	mathrand "math/rand"
	"net"
	"strconv"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
)

const (
	// MaxFrameSize is set based on max block size expected.
	MaxFrameSize = 500000
)

// ErrExceedMaxLen is the error thrown if the message size exceeds the max
// frame length.
var ErrExceedMaxLen = errors.New("message size exceeds max frame length")

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

// classifyDistance calculates floor of log2 of the distance between two nodes.
// As per that, classifyDistance returns rank of most significant bit in LE format.
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

// ComputeDistance returns both bucket number and XOR distance between two Peers.
func ComputeDistance(peer1, peer2 encoding.PeerInfo) (uint16, [16]byte) {
	return idXor(peer1.ID, peer2.ID)
}

// computePeerDummyID is helpful on simplifying ID on local net.
/* func computePeerDummyID(ip [4]byte, port uint16) [16]byte {
	var id [16]byte
	port -= 10000
	seed := make([]byte, 16)
	binary.LittleEndian.PutUint16(seed, port)
	copy(id[:], seed[0:16])
	return id
}
*/

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

// ------------------ NET UTILS ------------------ //

// GetOutboundIP returns local address.
func GetOutboundIP() net.IP {
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

// Format the UDP address, the UDP listener binds on.
func getLocalUDPAddress(port int) net.UDPAddr {
	laddr := net.UDPAddr{IP: GetOutboundIP()}
	laddr.Port = port
	return laddr
}

func readTCPFrame(r io.Reader) ([]byte, error) {
	// Read frame length.
	ln := make([]byte, 4)

	_, err := io.ReadFull(r, ln)
	if err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(ln[:])
	if length > MaxFrameSize {
		return nil, ErrExceedMaxLen
	}

	// Read packet payload
	payload := make([]byte, length)
	if _, err = io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func writeTCPFrame(w io.Writer, data []byte) error {
	frameLength := uint32(len(data))
	if frameLength > MaxFrameSize {
		return ErrExceedMaxLen
	}

	// Add packet length
	frame := new(bytes.Buffer)

	var b [4]byte

	binary.LittleEndian.PutUint32(b[:], frameLength)

	if _, err := frame.Write(b[:]); err != nil {
		return err
	}

	// Append packet payload
	if _, err := frame.Write(data); err != nil {
		return err
	}

	// Write data stream
	if _, err := w.Write(frame.Bytes()); err != nil {
		return err
	}

	return nil
}

// tcpSend Opens a TCP connection with the peer sent on the params and transmits
// a stream of bytes. Once transmitted, closes the connection.
func tcpSend(raddr net.UDPAddr, data []byte) {
	address := raddr.IP.String() + ":" + strconv.Itoa(raddr.Port)

	conn, err := net.Dial("tcp4", address)
	if err != nil {
		log.WithError(err).Warnf("Could not establish a peer connection %s.", raddr.String())
		return
	}

	log.WithField("src", conn.LocalAddr().String()).
		WithField("dest", raddr.String()).Traceln("Sending tcp")

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Write our message to the connection.
	if err = writeTCPFrame(conn, data); err != nil {
		log.WithError(err).Warnf("Could not write to addr %s", raddr.String())
	}

	_ = conn.Close()
}

// Gets the local address of the sender `Peer` and the UDPAddress of the
// receiver `Peer` and sends to it a UDP Packet with the payload inside.
func sendUDPPacket(laddr, raddr net.UDPAddr, payload []byte) {
	log.WithField("dest", raddr.String()).Tracef("Dialing udp")

	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0

	conn, err := net.DialUDP("udp", &laddr, &raddr)
	if err != nil {
		log.WithError(err).Warn("Could not establish a connection with the dest Peer.")
		return
	}

	log.WithField("src", conn.LocalAddr().String()).
		WithField("dest", raddr.String()).Traceln("Sending udp")

	// Simple write
	_, err = conn.Write(payload)
	if err != nil {
		log.WithError(err).Warn("Error while writing to the filedescriptor.")
	}

	_ = conn.Close()
}

// getRandDelegates selects n random and distinct items from `in` and
// copy them into `out` slice (no duplicates).
func getRandDelegates(beta uint8, in []encoding.PeerInfo, out *[]encoding.PeerInfo) error {
	if in == nil || out == nil {
		return errors.New("invalid in/out params")
	}

	if len(in) == 0 || len(*out) == int(beta) {
		return nil
	}

	maxval := int64(len(in))

	nBig, err := rand.Int(rand.Reader, big.NewInt(maxval))
	if err != nil {
		panic(err)
	}

	n := nBig.Int64()
	ind := uint32(n)

	*out = append(*out, in[ind])

	in[ind] = in[len(in)-1]
	in = in[:len(in)-1]

	return getRandDelegates(beta, in, out)
}

// getRandDelegatesByShuffle selects n random and distinct items from `in` and
// return them into `out` slice (no duplicates).
// For the performance benefit, this is based on pseudo-randomizes.
func getRandDelegatesByShuffle(beta uint8, in []encoding.PeerInfo) []encoding.PeerInfo {
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(in), func(i, j int) { in[i], in[j] = in[j], in[i] })

	out := make([]encoding.PeerInfo, 0, beta)

	for i := 0; i < int(beta); i++ {
		if i >= len(in) {
			break
		}

		out = append(out, in[i])
	}

	return out
}

func makeHeader(t byte, rt *RoutingTable) encoding.Header {
	return encoding.Header{
		MsgType:         t,
		RemotePeerID:    rt.LpeerInfo.ID,
		RemotePeerNonce: rt.localPeerNonce,
		RemotePeerPort:  rt.LpeerInfo.Port,
	}
}
