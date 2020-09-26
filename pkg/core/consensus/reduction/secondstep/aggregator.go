package secondstep

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

// The aggregator acts as a de facto storage unit for Reduction messages. Any message
// it receives will be aggregated into a StepVotes struct, organized by block hash.
// Once the key set for a StepVotes of a certain block hash reaches quorum, this
// StepVotes is passed on to the Reducer by use of the `haltChan` channel.
// An aggregator should be instantiated on a per-step basis and is no longer usable
// after reaching quorum and sending on `haltChan`.
type aggregator struct {
	handler *reduction.Handler

	voteSets map[string]struct {
		*message.StepVotes
		sortedset.Cluster
	}
}

// newAggregator returns an instantiated aggregator, ready for use.
func newAggregator(handler *reduction.Handler) *aggregator {

	return &aggregator{
		handler: handler,
		voteSets: make(map[string]struct {
			*message.StepVotes
			sortedset.Cluster
		}),
	}
}

// Collect a Reduction message, and add it's sender public key and signature to the
// StepVotes/Set kept under the corresponding block hash.
func (a *aggregator) collectVote(ev message.Reduction) (*result, error) {
	hdr := ev.State()
	hash := string(hdr.BlockHash)
	sv, found := a.voteSets[hash]
	if !found {
		sv.StepVotes = message.NewStepVotes()
		sv.Cluster = sortedset.NewCluster()
	}

	if err := sv.StepVotes.Add(ev.SignedHash, hdr.PubKeyBLS, hdr.Step); err != nil {
		lg.
			WithError(err).
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			WithField("quorum", sv.Cluster.TotalOccurrences()).
			Debug("secondstep, StepVotes.Add failed")
		return nil, err
	}

	votes := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)
	for i := 0; i < votes; i++ {
		sv.Cluster.Insert(hdr.PubKeyBLS)
	}
	a.voteSets[hash] = sv
	if sv.Cluster.TotalOccurrences() >= a.handler.Quorum(hdr.Round) {
		lg.
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			WithField("quorum", sv.Cluster.TotalOccurrences()).
			Debug("secondstep_quorum_reached")
		a.addBitSet(sv.StepVotes, sv.Cluster, hdr.Round, hdr.Step)
		return &result{
			Hash: hdr.BlockHash,
			SV:   *sv.StepVotes,
		}, nil

	}

	lg.
		WithField("round", hdr.Round).
		WithField("step", hdr.Step).
		WithField("quorum", sv.Cluster.TotalOccurrences()).
		Debug("secondstep_quorum_not_reached")
	return nil, nil
}

func (a *aggregator) addBitSet(sv *message.StepVotes, cluster sortedset.Cluster, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(cluster.Set)
}
