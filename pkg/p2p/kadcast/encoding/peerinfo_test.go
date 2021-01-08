// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPeerMarshaling(t *testing.T) {

	var id [16]byte
	seed, _ := crypto.RandEntropy(16)
	copy(id[:], seed[:])

	p := PeerInfo{
		[4]byte{127, 0, 0, 1},
		1234,
		id}

	var buf bytes.Buffer
	if err := p.MarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	var p2 PeerInfo
	if err := p2.UnmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	if !p.IsEqual(p2) {
		t.Error("marshal/unmarshal peer tuple failed")
	}
}

func TestPeerIsEqual(t *testing.T) {

	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{1, 2, 3, 4}
	var port uint16 = 9876

	p1 := PeerInfo{ip, port, id}
	p2 := PeerInfo{ip, port, id}

	if !p1.IsEqual(p2) {
		t.Error("expect they are equal")
	}

	p2.Port = 0
	if p1.IsEqual(p2) {
		t.Error("expect they are not equal")
	}
}
