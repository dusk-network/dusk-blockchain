package committee

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

const committeeSize = 64

type Agreement struct {
	*Extractor
	committees []user.VotingCommittee
}

func NewAgreement() *Agreement {
	return &Agreement{
		Extractor: NewExtractor(),
	}
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (a *Agreement) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size())
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (a *Agreement) Quorum() int {
	return int(float64(a.size()) * 0.75)
}

func (a *Agreement) size() int {
	return len(a.Extractor.Stakers.Provisioners.Members)
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (a *Agreement) Pack(set sortedset.Set, round uint64, step uint8) uint64 {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size())
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (a *Agreement) Unpack(bitset uint64, round uint64, step uint8) sortedset.Set {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size())
	return votingCommittee.Intersect(bitset)
}
