package firststep

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	log "github.com/sirupsen/logrus"
)

type aggregator struct {
	requestHalt func([]byte, ...*agreement.StepVotes)
	publisher   eventbus.Publisher
	handler     *reduction.Handler
	rpcBus      *rpcbus.RPCBus

	lock     sync.RWMutex
	voteSets map[string]struct {
		*agreement.StepVotes
		sortedset.Set
	}
}

func newAggregator(requestHalt func([]byte, ...*agreement.StepVotes), publisher eventbus.Publisher, handler *reduction.Handler, rpcBus *rpcbus.RPCBus) *aggregator {
	return &aggregator{
		requestHalt: requestHalt,
		publisher:   publisher,
		handler:     handler,
		rpcBus:      rpcBus,
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
		blockHash := hdr.BlockHash

		if err := verifyCandidateBlock(a.rpcBus, blockHash); err != nil {
			blockHash = emptyHash[:]
		}

		a.requestHalt(blockHash, sv.StepVotes)
	}
	return nil
}

func (a *aggregator) addBitSet(sv *agreement.StepVotes, set sortedset.Set, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(set)
}

func verifyCandidateBlock(rpcBus *rpcbus.RPCBus, blockHash []byte) error {
	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(blockHash, emptyHash[:]) {
		req := rpcbus.NewRequest(*(bytes.NewBuffer(blockHash)))
		if _, err := rpcBus.Call(rpcbus.VerifyCandidateBlock, req, 5*time.Second); err != nil {
			log.WithFields(log.Fields{
				"process": "reduction",
				"error":   err,
			}).Errorln("verifying the candidate block failed")
			return err
		}
	}

	return nil
}
