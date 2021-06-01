// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestWriteTCPFrame(t *testing.T) {
	writePayload := bytes.NewBufferString("some payload data")

	conn := &bytes.Buffer{}
	if err := writeTCPFrame(conn, writePayload.Bytes()); err != nil {
		t.Error(err)
	}

	readPayload, err := readTCPFrame(conn)
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

	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(MaxFrameSize+1))
	_, _ = frame.Write(b[:])

	readPayload, err := readTCPFrame(frame)
	if readPayload != nil {
		t.Error("expect an nil payload ")
	}

	if err != ErrExceedMaxLen {
		t.Error("expect frame size error")
	}
}

// TestClassifyDistance compares classifyDistance result with the result of the
// floor of log2 calculation over 64bits distances.
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

func TestGetRandDelegates(t *testing.T) {
	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{1, 2, 3, 4}

	in := make([]encoding.PeerInfo, 10)
	for i := 0; i < len(in); i++ {
		in[i] = encoding.PeerInfo{
			IP:   ip,
			Port: uint16(i),
			ID:   id,
		}
	}

	for i := 0; i < len(in); i++ {
		var beta uint8 = uint8(i)

		out := make([]encoding.PeerInfo, 0)
		getRandDelegates(beta, in, &out)

		if len(out) != int(beta) {
			t.Errorf("could not manage to generate %d delegates", beta)
		}
	}

	beta := uint8(len(in) * 2)
	out := make([]encoding.PeerInfo, 0)
	getRandDelegates(beta, in, &out)

	if len(out) != len(in) {
		t.Error("could not manage to generate n delegates")
	}
}

func TestGetRandDelegatesByShuffle(t *testing.T) {
	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{1, 2, 3, 4}

	in := make([]encoding.PeerInfo, 10)
	for i := 0; i < len(in); i++ {
		in[i] = encoding.PeerInfo{
			IP:   ip,
			Port: uint16(i),
			ID:   id,
		}
	}

	for i := 0; i < len(in); i++ {
		var beta uint8 = uint8(i)

		out := getRandDelegatesByShuffle(beta, in)

		if len(out) != int(beta) {
			t.Errorf("could not manage to generate %d delegates", beta)
		}
	}

	beta := uint8(len(in) * 2)
	out := getRandDelegatesByShuffle(beta, in)

	if len(out) != len(in) {
		t.Error("could not manage to generate n delegates")
	}
}
