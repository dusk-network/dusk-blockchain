// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPacketMarshalBinary(t *testing.T) {
	block, err := crypto.RandEntropy(1000)
	if err != nil {
		t.Fatal(err)
	}

	objectID := block[0:8]
	p := newPacket(objectID, 4, 3, 21, 222, block)

	var buf []byte

	if buf, err = p.marshal(); err != nil {
		t.Error(err)
	}

	p2 := Packet{}
	if err := p2.unmarshal(buf); err != nil {
		t.Error(err)
	}

	if p.blockID != p2.blockID {
		t.Fatal("blockID not equal")
	}

	if p.PaddingSize != p2.PaddingSize {
		t.Fatal("PaddingSize not equal")
	}

	if p.NumSourceSymbols != p2.NumSourceSymbols {
		t.Fatal("NumSourceSymbols not equal")
	}

	if p.transferLength != p2.transferLength {
		t.Fatal("transferLength not equal")
	}

	if !bytes.Equal(p.block[:], p2.block[:]) {
		t.Fatal("block not equal")
	}

	if !bytes.Equal(p.messageID[:], p2.messageID[:]) {
		t.Fatal("objectID not equal")
	}
}
