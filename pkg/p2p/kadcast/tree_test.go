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

	myPeer := encoding.PeerInfo{[4]byte{}, port, id}

	for port := 0; port <= 15; port++ {
		seed := make([]byte, 16)
		binary.LittleEndian.PutUint16(seed, uint16(port))
		var id [16]byte
		copy(id[:], seed[0:16])

		p := encoding.PeerInfo{[4]byte{}, uint16(port), id}
		tree.addPeer(myPeer, p)
	}

	t.Log(tree.trace(myPeer))
}
