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
func (p *agreementCommittee) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := p.UpsertCommitteeCache(round, step, p.size())
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (p *agreementCommittee) Quorum() int {
	return int(float64(p.size()) * 0.75)
}

func (p *agreementCommittee) size() int {
	provisioners := p.Provisioners()
	if provisioners.Size() > committeeSize {
		return committeeSize
	}
	return provisioners.Size()
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (p *agreementCommittee) Pack(set sortedset.Set, round uint64, step uint8) uint64 {
	votingCommittee := p.UpsertCommitteeCache(round, step, p.size())
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (p *agreementCommittee) Unpack(bitset uint64, round uint64, step uint8) sortedset.Set {
	votingCommittee := p.UpsertCommitteeCache(round, step, p.size())
	return votingCommittee.Intersect(bitset)
}
