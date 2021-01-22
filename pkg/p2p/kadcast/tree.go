// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
)

// Tree stores `L` buckets inside of it.
// This is basically the routing info of every peer.
type Tree struct {
	buckets [128]bucket
	mu      sync.RWMutex
}

// Allocates space for a tree and returns an empty intance of it.
func makeTree() Tree {
	var bucketList [128]bucket
	for i := 0; i < 128; i++ {
		bucketList[i] = makeBucket(uint8(i))
	}
	return Tree{
		buckets: bucketList,
	}
}

// Classifies and adds a Peer to the routing storage tree.
func (tree *Tree) addPeer(myPeer encoding.PeerInfo, otherPeer encoding.PeerInfo) {
	// routing state should not include myPeer
	if myPeer.IsEqual(otherPeer) {
		return
	}

	idl, _ := ComputeDistance(myPeer, otherPeer)

	// addPeer allows adding any peer with distance higher than 0. It allows a
	// peer with distance 0 only if it differs from myPeer. This is the
	// neighbor peer from the spanning tree myPeer belongs to.

	tree.mu.Lock()
	tree.buckets[idl].addPeer(otherPeer)
	tree.mu.Unlock()
}

// Returns the total amount of peers that a `Peer` is connected to.
func (tree *Tree) getTotalPeers() uint64 {
	var count uint64 = 0

	tree.mu.RLock()
	defer tree.mu.RUnlock()

	for _, bucket := range tree.buckets {
		count += uint64(len(bucket.entries))
	}

	return count
}

func (tree *Tree) trace(myPeer encoding.PeerInfo) string {
	logMsg := fmt.Sprintf("this_peer: %s, bucket peers num %d\n", myPeer.String(), tree.getTotalPeers())

	for _, b := range tree.buckets {
		for _, p := range b.entries {
			_, dist := ComputeDistance(myPeer, p)
			logMsg += fmt.Sprintf("bucket: %d, peer: %s, distance: %s\n", b.idLength, p.String(), hex.EncodeToString(dist[:]))
		}
	}

	return logMsg
}
