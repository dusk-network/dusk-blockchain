package secondstep

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type aggregator struct {
	requestHalt    func([]byte, ...*agreement.StepVotes)
	publisher      eventbus.Publisher
	handler        *reduction.Handler
	signer         consensus.Signer
	firstStepVotes *agreement.StepVotes

	lock     sync.RWMutex
	voteSets map[string]struct {
		*agreement.StepVotes
		sortedset.Set
	}
}

func newAggregator(requestHalt func([]byte, ...*agreement.StepVotes), publisher eventbus.Publisher, handler *reduction.Handler, firstStepVotes *agreement.StepVotes, signer consensus.Signer) *aggregator {
	return &aggregator{
		requestHalt:    requestHalt,
		publisher:      publisher,
		handler:        handler,
		signer:         signer,
		firstStepVotes: firstStepVotes,
		voteSets: make(map[string]struct {
			*agreement.StepVotes
			sortedset.Set
		}),
	}
}

func (a *aggregator) collectVote(ev reduction.Reduction, hdr header.Header) error {
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
	if len(sv.Set) == a.handler.Quorum() {
		a.addBitSet(sv.StepVotes, sv.Set, hdr.Round, hdr.Step)
		a.requestHalt(hdr.BlockHash, a.firstStepVotes, sv.StepVotes)
	}
	return nil
}

func (a *aggregator) addBitSet(sv *agreement.StepVotes, set sortedset.Set, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(set)
}
