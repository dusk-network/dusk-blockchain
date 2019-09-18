package committee

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type (
	// Committee is the interface for operations depending on the set of Provisioners
	// extracted for a given step
	Committee interface {
		IsMember(user.Provisioners, []byte, uint64, uint8) bool
		Quorum(user.Provisioners) int
	}

	// Foldable represents a Committee which can be packed into a bitset, to drastically
	// decrease the size needed for committee representation over the wire.
	Foldable interface {
		Committee
		Pack(user.Provisioners, sortedset.Set, uint64, uint8) uint64
		Unpack(user.Provisioners, uint64, uint64, uint8) sortedset.Set
	}

	// Cache is a wrapper around the Stakers struct, and contains the phase-specific
	// information, as well as a voting committee cache. It calls methods on the
	// Stakers, passing its own parameters to extract the desired info for a specific
	// phase.
	Cache struct {
		round          uint64
		committeeCache map[uint8]user.VotingCommittee
	}
)

// NewCache returns a committee extractor which maintains it's own cache.
// The extractor will still have an empty Stakers field, which should be populated by
// information received on round updates.
func NewCache() *Cache {
	return &Cache{
		committeeCache: make(map[uint8]user.VotingCommittee),
	}
}

// PregenerateCommittees will generate committees for a given range of steps.
// This method is best called on a round update, allowing the user to
// do some of the calculation before any events come in.
func (e *Cache) PregenerateCommittees(provisioners user.Provisioners, round uint64, initialStep, stepAmount uint8, size int) {
	for i := initialStep; i <= initialStep+stepAmount; i++ {
		e.UpsertCommitteeCache(provisioners, round, i, size)
	}
}

// UpsertCommitteeCache will return a voting committee for a given round, step and size.
// If the committee has not yet been produced before, it is put on the cache. If it has,
// it is simply retrieved and returned.
func (e *Cache) UpsertCommitteeCache(provisioners user.Provisioners, round uint64, step uint8, size int) user.VotingCommittee {
	if round > e.round {
		e.round = round
		e.committeeCache = make(map[uint8]user.VotingCommittee)
	}
	votingCommittee, found := e.committeeCache[step]
	if !found {
		votingCommittee = provisioners.CreateVotingCommittee(round, step, size)
		e.committeeCache[step] = votingCommittee
	}
	return votingCommittee
}
