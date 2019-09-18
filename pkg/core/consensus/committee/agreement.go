package committee

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

const committeeSize = 64

type Agreement struct {
	*Cache
	committees []user.VotingCommittee
}

func NewAgreement() *Agreement {
	return &Agreement{
		Cache: NewCache(),
	}
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (a *Agreement) IsMember(stakers user.Stakers, pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := a.UpsertCommitteeCache(stakers, round, step, a.size(stakers))
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (a *Agreement) Quorum(stakers user.Stakers) int {
	return int(float64(a.size(stakers)) * 0.75)
}

func (a *Agreement) size(stakers user.Stakers) int {
	size := len(stakers.Provisioners.Members)
	if size > committeeSize {
		return committeeSize
	}

	return size
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (a *Agreement) Pack(stakers user.Stakers, set sortedset.Set, round uint64, step uint8) uint64 {
	votingCommittee := a.UpsertCommitteeCache(stakers, round, step, a.size(stakers))
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (a *Agreement) Unpack(stakers user.Stakers, bitset uint64, round uint64, step uint8) sortedset.Set {
	votingCommittee := a.UpsertCommitteeCache(stakers, round, step, a.size(stakers))
	return votingCommittee.Intersect(bitset)
}
