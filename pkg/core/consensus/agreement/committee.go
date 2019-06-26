package agreement

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

const committeeSize = 100

type agreementCommittee struct {
	*committee.Extractor
}

func newAgreementCommittee(eventBroker wire.EventBroker) *agreementCommittee {
	return &agreementCommittee{
		Extractor: committee.NewExtractor(eventBroker),
	}
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (a *agreementCommittee) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size(round))
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (a *agreementCommittee) Quorum(round uint64) int {
	return int(float64(a.size(round)) * 0.75)
}

func (a *agreementCommittee) size(round uint64) int {
	provisioners := a.Provisioners()
	if provisioners.Size(round) > committeeSize {
		return committeeSize
	}
	return provisioners.Size(round)
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (a *agreementCommittee) Pack(set sortedset.Set, round uint64, step uint8) uint64 {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size(round))
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (a *agreementCommittee) Unpack(bitset uint64, round uint64, step uint8) sortedset.Set {
	votingCommittee := a.UpsertCommitteeCache(round, step, a.size(round))
	return votingCommittee.Intersect(bitset)
}
