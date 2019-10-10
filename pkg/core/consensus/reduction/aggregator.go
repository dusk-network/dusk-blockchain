package reduction

import (
	"sync"

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

func (a *aggregator) collectVote(ev Reduction, hdr header.Header) {
	a.lock.Lock()
	defer a.lock.Unlock()

	sv, found := a.voteSets[hdr.BlockHash]
	if !found {
		sv.StepVotes = NewStepVotes()
		sv.Set = sortedset.New()
	}

	if err := sv.StepVotes.Add(ev.SignedHash, hdr.PubKeyBLS, hdr.Step); err != nil {
		return err
	}

	sv.Set.Insert(hdr.PubKeyBLS)
	stepVotesMap[hdr.BlockHash] = sv
	if len(sv.Set) == a.r.handler.Quorum() {
		a.addBitSet(sv.StepVotes, hdr.Round, hdr.Step)
		r.addStepVotes(sv.StepVotes, hdr.BlockHash)
	}
}

func (a *aggregator) addBitSet(sv *agreement.StepVotes, set sortedset.Set, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(set)
}
