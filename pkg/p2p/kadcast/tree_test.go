package kadcast

import (
	"encoding/binary"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
)

func TestTree(t *testing.T) {

	tree := makeTree()

	var port uint16 = 7
	seed := make([]byte, 16)
	binary.LittleEndian.PutUint16(seed, port)
	var id [16]byte
	copy(id[:], seed[0:16])

	myPeer := encoding.PeerInfo{IP: [4]byte{},
		Port: port,
		ID:   id}

	for port := 0; port <= 15; port++ {
		seed := make([]byte, 16)
		binary.LittleEndian.PutUint16(seed, uint16(port))
		var id [16]byte
		copy(id[:], seed[0:16])

		p := encoding.PeerInfo{IP: [4]byte{},
			Port: uint16(port),
			ID:   id}

		tree.addPeer(myPeer, p)
	}

	t.Log(tree.trace(myPeer))
}
