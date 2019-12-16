package secondstep

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

// The aggregator acts as a de facto storage unit for Reduction messages. Any message
// it receives will be aggregated into a StepVotes struct, organized by block hash.
// Once the key set for a StepVotes of a certain block hash reaches quorum, this
// StepVotes is passed on to the Reducer by use of the `requestHalt` callback.
// An aggregator should be instantiated on a per-step basis and is no longer usable
// after reaching quorum and calling `requestHalt`.
type aggregator struct {
	requestHalt    func([]byte, ...*agreement.StepVotes)
	handler        *reduction.Handler
	firstStepVotes *agreement.StepVotes
	finished       bool

	lock     sync.RWMutex
	voteSets map[string]struct {
		*agreement.StepVotes
		sortedset.Cluster
	}
}

// newAggregator returns an instantiated aggregator, ready for use.
func newAggregator(
	requestHalt func([]byte, ...*agreement.StepVotes),
	handler *reduction.Handler,
	firstStepVotes *agreement.StepVotes) *aggregator {

	return &aggregator{
		requestHalt:    requestHalt,
		handler:        handler,
		firstStepVotes: firstStepVotes,
		voteSets: make(map[string]struct {
			*agreement.StepVotes
			sortedset.Cluster
		}),
	}
}

// Collect a Reduction message, and add it's sender public key and signature to the
// StepVotes/Set kept under the corresponding block hash.
func (a *aggregator) collectVote(ev reduction.Reduction, hdr header.Header) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.finished {
		return nil
	}

	hash := string(hdr.BlockHash)
	sv, found := a.voteSets[hash]
	if !found {
		sv.StepVotes = agreement.NewStepVotes()
		sv.Cluster = sortedset.NewCluster()
	}

	if err := sv.StepVotes.Add(ev.SignedHash, hdr.PubKeyBLS, hdr.Step); err != nil {
		return err
	}

	votes := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)
	for i := 0; i < votes; i++ {
		sv.Cluster.Insert(hdr.PubKeyBLS)
	}
	a.voteSets[hash] = sv
	if sv.Cluster.TotalOccurrences() >= a.handler.Quorum(hdr.Round) {
		a.finished = true
		a.addBitSet(sv.StepVotes, sv.Cluster, hdr.Round, hdr.Step)
		a.requestHalt(hdr.BlockHash, a.firstStepVotes, sv.StepVotes)
	}
	return nil
}

func (a *aggregator) addBitSet(sv *agreement.StepVotes, cluster sortedset.Cluster, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(cluster.Set)
}
