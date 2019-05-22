package reduction

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reputation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

const committeeSize = 50

type Reducers interface {
	committee.Committee
	reputation.Filter
}

type reductionCommittee struct {
	*committee.Extractor
}

func newReductionCommittee(eventBroker wire.EventBroker) *reductionCommittee {
	return &reductionCommittee{
		Extractor: committee.NewExtractor(eventBroker),
	}
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (p *reductionCommittee) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := p.UpsertCommitteeCache(round, step, p.size())
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (p *reductionCommittee) Quorum() int {
	return int(float64(p.size()) * 0.75)
}

func (p *reductionCommittee) size() int {
	provisioners := p.Provisioners()
	if provisioners.Size() > committeeSize {
		return committeeSize
	}
	return provisioners.Size()
}

func (p *reductionCommittee) FilterAbsentees(evs []wire.Event, round uint64, step uint8) user.VotingCommittee {
	votingCommittee := p.UpsertCommitteeCache(round, step, p.size())
	for _, ev := range evs {
		votingCommittee.Remove(ev.Sender())
	}
	return votingCommittee
}
