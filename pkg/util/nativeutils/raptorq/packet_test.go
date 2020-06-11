// +build

package raptorq

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPacketMarshalBinary(t *testing.T) {

	symbolData, err := crypto.RandEntropy(11111)
	if err != nil {
		t.Fatal(err)
	}

	sourceObjectID := symbolData[0:8]
	p := NewPacket(sourceObjectID, 4, 3, symbolData, 21, 222)

	var buf bytes.Buffer
	if err := p.MarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	p2 := Packet{}
	if err := p2.UnmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	if p != p2 {
		t.Fatal("not equal")
	}

	if !bytes.Equal(p.symbolData[:], p2.symbolData[:]) {
		t.Fatal("symbolData not equal")
	}

	if !bytes.Equal(p.sourceObjectID[:], p2.sourceObjectID[:]) {
		t.Fatal("sourceObjectID not equal")
	}
}
