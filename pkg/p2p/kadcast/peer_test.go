package kadcast

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestPeerMarshaling(t *testing.T) {

	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{}

	seed, _ := crypto.RandEntropy(16)
	copy(id[:], seed[:])

	p := Peer{ip, 1234, id}

	buf := marshalPeer(&p)
	p2 := unmarshalPeer(buf)

	if !bytes.Equal(p2.id[:], p.id[:]) {
		t.Error("ids not equal")
	}

	if !bytes.Equal(p2.ip[:], p.ip[:]) {
		t.Error("ip addresses not equal")
	}

	if p2.port != p.port {
		t.Error("port numbers not equal")
	}
}
