package committee

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type (
	// Committee is the interface for operations depending on the set of Provisioners
	// extracted for a given step
	Committee interface {
		IsMember([]byte, uint64, uint8) bool
		Quorum(uint64) int
		RemoveExpiredProvisioners(*bytes.Buffer) error
	}

	// Foldable represents a Committee which can be packed into a bitset, to drastically
	// decrease the size needed for committee representation over the wire.
	Foldable interface {
		Committee
		Pack(sortedset.Set, uint64, uint8) uint64
		Unpack(uint64, uint64, uint8) sortedset.Set
	}

	// Extractor is a wrapper around the Stakers struct, and contains the phase-specific
	// information, as well as a voting committee cache. It calls methods on the
	// Stakers, passing its own parameters to extract the desired info for a specific
	// phase.
	Extractor struct {
		user.Stakers
		round          uint64
		committeeCache map[uint8]user.VotingCommittee
	}
)

// NewExtractor returns a committee extractor which maintains it's own cache.
func NewExtractor() *Extractor {
	return &Extractor{
		committeeCache: make(map[uint8]user.VotingCommittee),
	}
}

// PregenerateCommittees will generate committees for a given amount of steps.
// This method is best called when the Stakers field is updated, allowing the user to
// do some of the calculation before any events come in.
func (e *Extractor) PregenerateCommittees(round, totalWeight uint64, steps uint8, size int) {
	for i := uint8(1); i <= steps; i++ {
		e.UpsertCommitteeCache(round, totalWeight, i, size)
	}
}

// UpsertCommitteeCache will return a voting committee for a given round, step and size.
// If the committee has not yet been produced before, it is put on the cache. If it has,
// it is simply retrieved and returned.
func (e *Extractor) UpsertCommitteeCache(round, totalWeight uint64, step uint8, size int) user.VotingCommittee {
	if round > e.round {
		e.round = round
		e.committeeCache = make(map[uint8]user.VotingCommittee)
	}
	votingCommittee, found := e.committeeCache[step]
	if !found {
		votingCommittee = e.Stakers.CreateVotingCommittee(round, totalWeight, step, size)
		e.committeeCache[step] = votingCommittee
	}
	return votingCommittee
}
