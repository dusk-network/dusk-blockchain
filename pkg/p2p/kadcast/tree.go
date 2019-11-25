package kadcast

// Tree stores `L` buckets inside of it.
// This is basically the routing info of every peer.
type Tree struct {
	buckets [128]Bucket
}

// Allocates space for a tree and returns an empty intance of it.
//
// It also sets our `Peer` info in the lowest order bucket.
func makeTree(myPeer Peer) Tree {
	var bucketList [128]Bucket
	for i := 0; i < 128; i++ {
		bucketList[i] = makeBucket(uint8(i))
	}
	// Add my `Peer` info on the lowest `Bucket`.
	bucketList[0].addPeerToBucket(myPeer)
	return Tree{
		buckets: bucketList,
	}
}

// Classifies and adds a Peer to the routing storage tree.
func (tree *Tree) addPeer(myPeer Peer, otherPeer Peer) {
	idl := myPeer.computePeerDistance(otherPeer)
	if idl == 0 {
		return
	}
	tree.buckets[idl].addPeerToBucket(otherPeer)
}

// Returns the total ammount of peers that a `Peer` is connected to.
func (tree *Tree) getTotalPeers() uint64 {
	var count uint64 = 0
	for i, bucket := range tree.buckets {
		if i != 0 {
			count += uint64(bucket.peerCount)
		}
	}
	return count
}
