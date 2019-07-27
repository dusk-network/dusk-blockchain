package reduction

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reputation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

const committeeSize = 64

// Reducers defines a committee of reducers, and provides the ability to detect those
// who are not properly participating in this phase of the consensus.
type Reducers interface {
	committee.Committee
	reputation.Filter
}

type reductionCommittee struct {
	*committee.Extractor
}

func newReductionCommittee(eventBroker wire.EventBroker, db database.DB) *reductionCommittee {
	return &reductionCommittee{
		Extractor: committee.NewExtractor(eventBroker, db),
	}
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (r *reductionCommittee) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := r.UpsertCommitteeCache(round, step, r.size(round))
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (r *reductionCommittee) Quorum(round uint64) int {
	return int(float64(r.size(round)) * 0.75)
}

func (r *reductionCommittee) size(round uint64) int {
	provisioners := r.Provisioners()
	size := provisioners.Size(round)
	if size > committeeSize {
		return committeeSize
	}
	return size
}

func (r *reductionCommittee) FilterAbsentees(evs []wire.Event, round uint64, step uint8) user.VotingCommittee {
	votingCommittee := r.UpsertCommitteeCache(round, step, r.size(round))
	for _, ev := range evs {
		votingCommittee.Remove(ev.Sender())
	}
	return votingCommittee
}
