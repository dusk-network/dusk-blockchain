package firststep

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// Helper for reducing test boilerplate
type Helper struct {
	*reduction.Helper
	StepVotesChan      chan message.Message
	lock               sync.RWMutex
	failOnFetching     bool
	failOnVerification bool
}

// NewHelper creates a Helper used for testing the first step Reducer.
// `startGoroutines` can be specified to simultaneously launch goroutines
// that intercept RPC calls made by the first step Reducer.
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, timeOut time.Duration, startGoroutines bool) *Helper {
	hlp := &Helper{
		Helper:             reduction.NewHelper(eb, rpcbus, provisioners, CreateReducer, timeOut),
		StepVotesChan:      make(chan message.Message, 1),
		failOnFetching:     false,
		failOnVerification: false,
	}

	if startGoroutines {
		go hlp.provideCandidateBlock()
		go hlp.processCandidateVerificationRequest()
	}
	hlp.createResultChan()
	return hlp
}

// FailOnVerification tells the RPC bus to return an error
func (hlp *Helper) FailOnVerification(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnVerification = flag
}

// FailOnFetching sets the failOnFetching flag
func (hlp *Helper) FailOnFetching(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnFetching = flag
}

func (hlp *Helper) shouldFailFetching() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	f := hlp.failOnFetching
	return f
}

func (hlp *Helper) shouldFailVerification() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	f := hlp.failOnVerification
	return f
}

func (hlp *Helper) provideCandidateBlock() {
	c := make(chan rpcbus.Request, 1)
	_ = hlp.RBus.Register(topics.GetCandidate, c)
	for {
		r := <-c
		if hlp.shouldFailFetching() {
			r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("could not get candidate block"))
			continue
		}

		r.RespChan <- rpcbus.NewResponse(message.Candidate{}, nil)
	}
}

func (hlp *Helper) processCandidateVerificationRequest() {
	v := make(chan rpcbus.Request, 1)
	if err := hlp.RBus.Register(topics.VerifyCandidateBlock, v); err != nil {
		panic(err)
	}
	for {
		r := <-v
		if hlp.shouldFailVerification() {
			r.RespChan <- rpcbus.NewResponse(nil, errors.New("verification failed"))
			continue
		}

		r.RespChan <- rpcbus.NewResponse(nil, nil)
	}
}

// CreateResultChan is used by tests (internal and external) to quickly wire the StepVotes resulting from the firststep reduction to a channel to listen to
func (hlp *Helper) createResultChan() {
	chanListener := eventbus.NewChanListener(hlp.StepVotesChan)
	hlp.Bus.Subscribe(topics.StepVotes, chanListener)
}

// ActivateReduction sends the reducer a BestScore event to trigger a EvenPlayer.Resume
func (hlp *Helper) ActivateReduction(hash []byte) {
	hlp.CollectionWaitGroup.Wait()
	hdr := header.Header{BlockHash: hash, Round: hlp.Round, Step: hlp.Step(), PubKeyBLS: hlp.PubKeyBLS}
	_ = hlp.Reducer.(*Reducer).CollectBestScore(hdr)
}

// NextBatch forwards additional batches of consensus.Event. It takes care of marshaling the right Step when creating the Signature
func (hlp *Helper) NextBatch() []byte {
	blockHash, _ := crypto.RandEntropy(32)
	hlp.ActivateReduction(blockHash)
	hlp.SendBatch(blockHash)
	return blockHash
}

// Kickstart a Helper without sending any reduction event, and without starting goroutines to
// intercept RPCBus calls.
func Kickstart(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, timeOut time.Duration) (*Helper, []byte) {
	h := NewHelper(eb, rpcbus, provisioners, timeOut, false)
	hash := kickstart(h)
	return h, hash
}

// KickstartConcurrent kickstarts a Helper without sending any reduction event, starting
// goroutines to intercept RPCBus calls.
func KickstartConcurrent(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, timeOut time.Duration) (*Helper, []byte) {
	h := NewHelper(eb, rpcbus, provisioners, timeOut, true)
	hash := kickstart(h)
	return h, hash
}

func kickstart(h *Helper) []byte {
	roundUpdate := consensus.MockRoundUpdate(h.Round, h.P)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	h.ActivateReduction(hash)
	return hash
}

// ProduceFirstStepVotes encapsulates the process of creating and forwarding Reduction events
func ProduceFirstStepVotes(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, timeOut time.Duration) (*Helper, []byte) {
	hlp, hash := KickstartConcurrent(eb, rpcbus, provisioners, timeOut)
	hlp.SendBatch(hash)
	return hlp, hash
}
