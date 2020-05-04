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

	p := PeerInfo{ip, 1234, id}

	buf := marshalPeerInfo(&p)
	p2 := unmarshalPeerInfo(buf)

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

func TestPeerIsEqual(t *testing.T) {

	ip := [4]byte{127, 0, 0, 1}
	id := [16]byte{1, 2, 3, 4}
	var port uint16 = 9876

	p1 := PeerInfo{ip, port, id}
	p2 := PeerInfo{ip, port, id}

	if !p1.IsEqual(p2) {
		t.Error("expect they are equal")
	}

	p2.port = 0
	if p1.IsEqual(p2) {
		t.Error("expect they are not equal")
	}
}
