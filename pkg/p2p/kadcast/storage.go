package kadcast

// MAX_BUCKET_PEERS represents the maximum
//number of peers that a `Bucket` can hold.
const MAX_BUCKET_PEERS uint8 = 25

// Bucket stores peer info of the peers that are at a certain
// distance range to the peer itself.
type Bucket struct {
	idLength         uint8
	peerCount        uint8
	totalPeersPassed uint64
	// Should always be less than `MAX_BUCKET_PEERS`
	entries []Peer
	// This map keeps the order of arrivals for LRU
	lru map[Peer]uint64
	// This map allows us to quickly see if a Peer is
	// included on a entries set without iterating over
	// it.
	lruPresent map[Peer]bool
}

// Allocates space for a `Bucket` and returns a instance
// of it with the specified `idLength`.
func makeBucket(idlen uint8) Bucket {
	return Bucket{
		idLength:         idlen,
		totalPeersPassed: 0,
		peerCount:        0,
		entries:          make([]Peer, 0, MAX_BUCKET_PEERS),
		lru:              make(map[Peer]uint64),
		lruPresent:       make(map[Peer]bool),
	}
}

// Adds a `Peer` to the `Bucket` entries list.
// It also increments the peerCount all according
// the LRU policy.
func (b *Bucket) addPeerToBucket(peer Peer) {
	// Check if the entries set can hold more peers
	if len(b.entries) < int(MAX_BUCKET_PEERS) {
		// Insert it into the set if not present
		// on the current entries set.
		if b.lruPresent[peer] == false {
			b.entries = append(b.entries, peer)
			b.peerCount++
		}
		// Store recently used peer.
		b.lru[peer] = b.totalPeersPassed
		b.totalPeersPassed++
	} else {
		// If the entries set is full, we perform
		// LRU and remove a peer to include the new one.
		//
		// Check if peer is not already present into the
		// entries set
		if b.lruPresent[peer] == false {
			panic("uninplemented")
		}
	}

}

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
