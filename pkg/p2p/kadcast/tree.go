package kadcast

// Tree stores `L` buckets inside of it.
// This is basically the routing info of every peer.
type Tree struct{ buckets [128]Bucket }

// Allocates space for a tree and returns an empty intance of it.
func makeTree() Tree {
	panic("unimplemented")
}

// Classifies and adds a Peer to the routing storage tree.
func (tree *Tree) addPeer(peer *Peer) {
	panic("unimplemeneted")
}
