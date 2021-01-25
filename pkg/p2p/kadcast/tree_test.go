// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"encoding/binary"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
)

func TestTree(t *testing.T) {
	var port uint16 = 7
	var id [16]byte

	tree := makeTree()

	seed := make([]byte, 16)
	binary.LittleEndian.PutUint16(seed, port)

	copy(id[:], seed[0:16])

	myPeer := encoding.PeerInfo{
		IP:   [4]byte{},
		Port: port,
		ID:   id,
	}

	for port := 0; port <= 15; port++ {
		seed := make([]byte, 16)
		binary.LittleEndian.PutUint16(seed, uint16(port))
		var id [16]byte
		copy(id[:], seed[0:16])

		p := encoding.PeerInfo{
			IP:   [4]byte{},
			Port: uint16(port),
			ID:   id,
		}

		tree.addPeer(myPeer, p)
	}

	t.Log(tree.trace(myPeer))
}
