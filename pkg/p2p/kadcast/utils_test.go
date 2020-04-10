package kadcast

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPOW(t *testing.T) {
	a := Peer{
		ip:   [4]byte{192, 169, 1, 1},
		port: 25519,
		id:   [16]byte{22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22},
	}

	println(a.computePeerNonce())
}

func TestWriteTCPFrame(t *testing.T) {
	writePayload := bytes.NewBufferString("some payload data")

	conn := &bytes.Buffer{}
	if err := writeTCPFrame(conn, writePayload.Bytes()); err != nil {
		t.Error(err)
	}

	readPayload, _, err := readTCPFrame(conn)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(readPayload, writePayload.Bytes()) {
		t.Error("payload read is invalid")
	}
}

func TestWriteTCPFrameErr(t *testing.T) {
	writePayload := make([]byte, MaxFrameSize+1)

	conn := &bytes.Buffer{}
	if err := writeTCPFrame(conn, writePayload); err != ErrExceedMaxLen {
		t.Error("expect frame size error")
	}
}

func TestReadTCPFrameErr(t *testing.T) {

	frame := new(bytes.Buffer)
	_ = encoding.WriteUint32LE(frame, MaxFrameSize+1)

	readPayload, _, err := readTCPFrame(frame)
	if readPayload != nil {
		t.Error("expect an nil payload ")
	}
	if err != ErrExceedMaxLen {
		t.Error("expect frame size error")
	}
}

// TestClassifyDistance compares classifyDistance result with the result of the
// floor of log2 calculation over 64bits distances
func TestClassifyDistance(t *testing.T) {

	c := 0
	for {

		b, _ := crypto.RandEntropy(8)
		var distance [16]byte
		copy(distance[:], b)

		result := float64(classifyDistance(distance))

		v := binary.LittleEndian.Uint64(distance[:])
		expected := math.Floor(math.Log2(float64(v)))

		if expected != result {
			t.Error("classifyDistance result not equal floor(log2(distance))")
		}

		if c++; c == 10 {
			break
		}
	}

	// check corner cases
	var distance [16]byte
	if classifyDistance(distance) != 0 {
		t.Errorf("invalid calculation on 0 distance")
	}
}

func TestGenerateRandomDelegates(t *testing.T) {

	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{1, 2, 3, 4}

	in := make([]Peer, 10)
	for i := 0; i < len(in); i++ {
		in[i] = Peer{ip, uint16(i), id}
	}

	for i := 0; i < len(in); i++ {
		var beta uint8 = uint8(i)
		out := make([]Peer, 0)
		generateRandomDelegates(beta, in, &out)

		if len(out) != int(beta) {
			t.Errorf("could not manage to generate %d delegates", beta)
		}
	}

	beta := uint8(len(in) * 2)
	out := make([]Peer, 0)
	generateRandomDelegates(beta, in, &out)

	if len(out) != len(in) {
		t.Error("could not manage to generate n delegates")
	}
}
