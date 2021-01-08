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

// TestMarshalHeader ensures both marshal/unmarshal header. In addition, it
// covers a test for computeNonce
func TestMarshalHeader(t *testing.T) {

	// Construct random header
	h := Header{
		MsgType:        PongMsg,
		RemotePeerPort: 1234,
		Reserved:       [2]byte{21, 22},
	}

	b, err := crypto.RandEntropy(IDLen)
	if err != nil {
		t.Fatal(err)
	}
	copy(h.RemotePeerID[:], b)

	h.RemotePeerNonce = ComputeNonce(h.RemotePeerID[:])

	var buf bytes.Buffer
	if err := h.MarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	var h2 Header
	if err := h2.UnmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	if h != h2 {
		t.Error("header marshal/unmarshal failed")
	}
}

func TestNodesPayloadMarshaling(t *testing.T) {
	var p NodesPayload

	peer := MakePeer([4]byte{192, 168, 1, 2}, 1234)
	p.Peers = append(p.Peers, peer)

	peer2 := MakePeer([4]byte{212, 222, 3, 3}, 5678)
	p.Peers = append(p.Peers, peer2)

	var buf bytes.Buffer
	if err := p.MarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	var p2 NodesPayload
	if err := p2.UnmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	for i := range p.Peers {
		if !p.Peers[i].IsEqual(p2.Peers[i]) {
			t.Error("invalid nodes payload marshaling")
		}
	}
}

func TestBroadcastPayloadMarshaling(t *testing.T) {

	b, err := crypto.RandEntropy(1000)
	if err != nil {
		t.Fatal(err)
	}

	p := BroadcastPayload{
		Height:      123,
		GossipFrame: b,
	}

	var buf bytes.Buffer
	if err := p.MarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	var p2 BroadcastPayload
	if err := p2.UnmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	if p.Height != p2.Height {
		t.Error("broadcast height wrongly marshaled")
	}

	if !bytes.Equal(p.GossipFrame, p2.GossipFrame) {
		t.Error("broadcast gossipFrame wrongly marshaled")
	}
}
