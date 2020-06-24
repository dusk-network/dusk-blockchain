package rcudp

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPacketMarshalBinary(t *testing.T) {

	block, err := crypto.RandEntropy(11111)
	if err != nil {
		t.Fatal(err)
	}

	objectID := block[0:8]
	p := newPacket(objectID, 4, 3, 21, 222, block)

	var buf bytes.Buffer
	if err := p.marshalBinary(&buf); err != nil {
		t.Error(err)
	}

	p2 := Packet{}
	if err := p2.unmarshalBinary(&buf); err != nil {
		t.Error(err)
	}

	if p != p2 {
		t.Fatal("not equal")
	}

	if !bytes.Equal(p.block[:], p2.block[:]) {
		t.Fatal("block not equal")
	}

	if !bytes.Equal(p.messageID[:], p2.messageID[:]) {
		t.Fatal("objectID not equal")
	}

}
