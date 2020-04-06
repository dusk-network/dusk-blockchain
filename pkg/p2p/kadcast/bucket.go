package kadcast

import (
	"errors"
	"math/rand"
)

// MaxBucketPeers represents the maximum
//number of peers that a `bucket` can hold.
var MaxBucketPeers uint8 = 25

// bucket stores peer info of the peers that are at a certain
// distance range to the peer itself.
type bucket struct {
	idLength         uint8
	peerCount        uint8
	totalPeersPassed uint64
	// Should always be less than `MaxBucketPeers`
	entries []Peer
	// This map keeps the order of arrivals for LRU
	lru map[Peer]uint64
	// This map allows us to quickly see if a Peer is
	// included on a entries set without iterating over
	// it.
	lruPresent map[Peer]bool
}

// Allocates space for a `bucket` and returns a instance
// of it with the specified `idLength`.
func makeBucket(idlen uint8) bucket {
	return bucket{
		idLength:         idlen,
		totalPeersPassed: 0,
		peerCount:        0,
		entries:          make([]Peer, 0, MaxBucketPeers),
		lru:              make(map[Peer]uint64),
		lruPresent:       make(map[Peer]bool),
	}
}

// Finds the Least Recently Used Peer on the entries set
// of the `bucket` and returns it's index on the entries
// set and the `Peer` info that is hold on it.
func (b bucket) findLRUPeerIndex() (int, uint64) {
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
// It also maps the `Peer` to false on the LRU map.
// The resulting slice of entries is then returned.
func (b *bucket) removePeerAtIndex(index int) []Peer {
	// Remove peer from the lruPresent map.
	b.lruPresent[b.entries[index]] = false

	b.entries[index] = b.entries[len(b.entries)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return b.entries[:len(b.entries)-1]
}

// Picks a random Peer from the bucket and returns it.
func (b bucket) getRandomPeer() (*Peer, error) {
	if b.peerCount == 0 {
		// If the bucket has no nodes inside, we return an error.
		return nil, errors.New("bucket empty")
	}
	maxVal := len(b.entries)
	return &b.entries[rand.Intn(maxVal)], nil
}

// Adds a `Peer` to the `bucket` entries list.
// It also increments the peerCount all according
// the LRU policy.
func (b *bucket) addPeer(peer Peer) {
	// Check if the entries set can hold more peers.
	if len(b.entries) < int(MaxBucketPeers) {
		// Insert it into the set if not present
		// on the current entries set.
		if !b.lruPresent[peer] {
			b.entries = append(b.entries, peer)
			b.peerCount++
			b.lruPresent[peer] = true
		}
		// Store recently used peer.
		b.lru[peer] = b.totalPeersPassed
		b.totalPeersPassed++
		return
	}
	// If the entries set is full, we perform
	// LRU and remove a peer to include the new one.
	//
	// Check if peer is not already present into the
	// entries set
	if !b.lruPresent[peer] {
		// Search for the least recently used peer.
		var index, _ = b.findLRUPeerIndex()
		// Remove it from the entries set and from
		// the lruPresent map.
		b.entries = b.removePeerAtIndex(index)
		// Add the new peer to the entries set.
		b.entries = append(b.entries, peer)
		b.lruPresent[peer] = true
		b.totalPeersPassed++
	}
	b.lru[peer] = b.totalPeersPassed
}
