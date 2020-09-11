package kadcast

import "sync"

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
func (tree *Tree) addPeer(myPeer Peer, otherPeer Peer) {

	// routing state should not include myPeer
	if myPeer.IsEqual(otherPeer) {
		return
	}

	idl, _ := myPeer.computeDistance(otherPeer)

	// addPeer allows adding any peer with distance higher than 0. It allows a
	// peer with distance 0 only if it differs from myPeer. This is the
	// neighbor peer from the spanning tree myPeer belongs to.

	tree.mu.Lock()
	defer tree.mu.Unlock()
	tree.buckets[idl].addPeer(otherPeer)
}

// Returns the total amount of peers that a `Peer` is connected to.
func (tree *Tree) getTotalPeers() uint64 {
	tree.mu.RLock()
	defer tree.mu.RUnlock()

	var count uint64 = 0
	for _, bucket := range tree.buckets {
		count += uint64(len(bucket.entries))
	}
	return count
}
