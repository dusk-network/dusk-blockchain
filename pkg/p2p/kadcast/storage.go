package kadcast

// Bucket stores peer info of the peers that are at a certain
// distance range to the peer itself.
type Bucket struct {
	length  uint8
	entries []Peer
}

// Tree stores `L` buckets inside of it.
// This is basically the routing info of every peer.
type Tree struct{ buckets [256]Bucket }

// Allocates space for a tree and returns an empty intance of it.
func makeTree() Tree {
	panic("unimplemented")
}

// Adds a Peer to the routing storage tree.
func (tree *Tree) addPeer(peer *Peer) {
	panic("unimplemeneted")
}
