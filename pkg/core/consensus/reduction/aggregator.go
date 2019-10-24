package reduction

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type aggregator struct {
	r *reducer

	lock     sync.RWMutex
	voteSets map[string]struct {
		*agreement.StepVotes
		sortedset.Set
	}
}

func newAggregator(r *reducer) *aggregator {
	return &aggregator{
		r: r,
		voteSets: make(map[string]struct {
			*agreement.StepVotes
			sortedset.Set
		}),
	}
}

func (a *aggregator) collectVote(ev Reduction, hdr header.Header) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	hash := string(hdr.BlockHash)
	sv, found := a.voteSets[hash]
	if !found {
		sv.StepVotes = agreement.NewStepVotes()
		sv.Set = sortedset.New()
	}

	if err := sv.StepVotes.Add(ev.SignedHash, hdr.PubKeyBLS, hdr.Step); err != nil {
		return err
	}

	sv.Set.Insert(hdr.PubKeyBLS)
	a.voteSets[hash] = sv
	if len(sv.Set) == a.r.handler.Quorum() {
		a.addBitSet(sv.StepVotes, sv.Set, hdr.Round, hdr.Step)
		a.r.addStepVotes(sv.StepVotes, hdr.BlockHash)
	}
	return nil
}

func (a *aggregator) addBitSet(sv *agreement.StepVotes, set sortedset.Set, round uint64, step uint8) {
	committee := a.r.handler.Committee(round, step)
	sv.BitSet = committee.Bits(set)
}
