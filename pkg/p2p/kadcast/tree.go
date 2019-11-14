package kadcast

import "net"

// Tree stores `L` buckets inside of it.
// This is basically the routing info of every peer.
type Tree struct {
	buckets [128]Bucket
}

// Allocates space for a tree and returns an empty intance of it.
//
// It also sets our `Peer` info.
func makeTree(myPeer *Peer) Tree {
	var bucketList [128]Bucket
	for i := 0; i < 128; i++ {
		bucketList[i] = makeBucket(uint8(i))
	}
	// Add my `Peer` info on the lowest `Bucket`.
	bucketList[0].addPeerToBucket(*myPeer)
	return Tree{
		buckets: bucketList,
	}
}

// Classifies and adds a Peer to the routing storage tree.
func (tree *Tree) addPeer(myPeer *Peer, otherPeer *Peer) {
	idl := myPeer.computePeerDistance(otherPeer)
	if idl == 0 {
		return
	}
	tree.buckets[idl].addPeerToBucket(*otherPeer)
}

// Returns the total ammount of peers that a `Peer` is connected to.
func (tree Tree) getTotalPeers() uint64 {
	var count uint64 = 0
	for i := 1; i < 128; i++ {
		count += uint64(tree.buckets[i].peerCount)
	}
	return count
}

// Grabs the networking info of the `Peer` owner of the 
// routing tree and returns it as `UDPAddr`.
func (tree Tree) getMyPeerUDPAddr() *net.UDPAddr {
	return tree.buckets[0].entries[0].getUDPAddr()
}
