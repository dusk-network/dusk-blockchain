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

// Finds the Least Recently Used Peer on the entries set
// of the `Bucket` and returns it's index on the entries
// set and the `Peer` info that is hold on it.
func (b Bucket) findLruPeerIndex() (int, uint64) {
	var val = b.totalPeersPassed
	i := 0
	for index, p := range b.entries {
		if b.lru[p] <= val {
			val = b.lru[p]
			i = index
		}
	}
	return i, val
}

// Remove a `Peer` from the entries set without
// caring about the order.
// The resulting slice of entries is then returned.
func (b Bucket) removePeerAtIndex(index int) []Peer {
	b.entries[index] = b.entries[len(b.entries)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return b.entries[:len(b.entries)-1]
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
			// Search for the least recently used peer.
			var index, _ = b.findLruPeerIndex()
			// Remove it from the entries set.
			b.entries = b.removePeerAtIndex(index)
			// Add the new peer to the entries set.
			b.entries = append(b.entries, peer)
			b.totalPeersPassed++
		}
		b.lru[peer] = b.totalPeersPassed
	}
}
