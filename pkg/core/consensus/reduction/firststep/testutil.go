package firststep

import (
	"bytes"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// Helper for reducing test boilerplate
type Helper struct {
	*reduction.Helper
	StepVotesChan      chan bytes.Buffer
	lock               sync.RWMutex
	failOnVerification bool
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int) *Helper {
	hlp := &Helper{
		Helper:             reduction.NewHelper(eb, rpcbus, provisioners, CreateReducer),
		StepVotesChan:      make(chan bytes.Buffer, 1),
		failOnVerification: false,
	}

	go hlp.verifyCandidateBlock()
	hlp.createResultChan()
	return hlp
}

// FailOnVerification tells the RPC bus to return an error
func (hlp *Helper) FailOnVerification(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnVerification = flag
}

func (hlp *Helper) shouldFail() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	f := hlp.failOnVerification
	return f
}

func (hlp *Helper) verifyCandidateBlock() {
	v := make(chan rpcbus.Request, 1)
	hlp.RBus.Register(rpcbus.VerifyCandidateBlock, v)
	for {
		r := <-v
		if hlp.shouldFail() {
			r.RespChan <- rpcbus.Response{bytes.Buffer{}, errors.New("verification failed")}
		} else {
			r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
		}
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
	hlp.Reducer.(*Reducer).CollectBestScore(consensus.Event{hdr, bytes.Buffer{}})
}

// NextBatch forwards additional batches of consensus.Event. It takes care of marshalling the right Step when creating the Signature
func (hlp *Helper) NextBatch() []byte {
	blockHash, _ := crypto.RandEntropy(32)
	hlp.ActivateReduction(blockHash)
	hlp.SendBatch(blockHash)
	return blockHash
}

// Kickstart a Helper without sending any reduction event
func Kickstart(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, nr int) (*Helper, []byte) {
	h := NewHelper(eb, rpcbus, nr)
	roundUpdate := consensus.MockRoundUpdate(h.Round, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	h.ActivateReduction(hash)
	return h, hash
}

// ProduceFirstStepVotes encapsulates the process of creating and forwarding Reduction events
func ProduceFirstStepVotes(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, nr int) (*Helper, []byte) {
	hlp, hash := Kickstart(eb, rpcbus, nr)
	hlp.SendBatch(hash)
	return hlp, hash
}
