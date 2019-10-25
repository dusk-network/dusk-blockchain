package firststep

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
)

// Helper for reducing test boilerplate
type Helper struct {
	*Factory
	Keys    []key.ConsensusKeys
	P       *user.Provisioners
	Reducer *Reducer
	stepper consensus.Stepper
	signer  consensus.Signer

	StepVotesChan chan bytes.Buffer
	nr            int

	lock               sync.RWMutex
	failOnVerification bool
	Handler            *reduction.Handler
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, stepper consensus.Stepper, signer consensus.Signer, provisioners int) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	factory := NewFactory(eb, rpcbus, keys[0], 100*time.Millisecond)
	a := factory.Instantiate()
	red := a.(*Reducer)
	hlp := &Helper{
		Factory:            factory,
		Keys:               keys,
		P:                  p,
		Reducer:            red,
		stepper:            stepper,
		signer:             signer,
		StepVotesChan:      make(chan bytes.Buffer, 1),
		nr:                 provisioners,
		failOnVerification: false,
		Handler:            reduction.NewHandler(keys[0], *p),
	}
	go hlp.verifyCandidateBlock()
	hlp.createResultChan()
	return hlp
}

func (hlp *Helper) Verify(hash []byte, sv *agreement.StepVotes) error {
	vc := hlp.P.CreateVotingCommittee(1, 1, hlp.nr)
	sub := vc.Intersect(sv.BitSet)
	apk, err := agreement.ReconstructApk(sub)
	if err != nil {
		return err
	}

	return header.VerifySignatures(1, 1, hash, apk, sv.Signature)
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
	hlp.RpcBus.Register(rpcbus.VerifyCandidateBlock, v)
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

// SendBatch of consensus events to the reducer callback CollectReductionEvent
func (hlp *Helper) SendBatch(hash []byte) {
	batch := hlp.Spawn(hash)
	for _, ev := range batch {
		go hlp.Reducer.CollectReductionEvent(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) []consensus.Event {
	evs := make([]consensus.Event, hlp.nr)
	vc := hlp.P.CreateVotingCommittee(1, 1, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := reduction.MockConsensusEvent(hash, 1, 1, hlp.Keys, vc, i)
		evs[i] = ev

	}
	return evs
}

// Initialize the reducer with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.Reducer.Initialize(hlp.stepper, hlp.signer, ru)
}

// ProduceFirstStepVotes encapsulates the process of creating and forwarding Reduction events
func ProduceFirstStepVotes(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, stepper consensus.Stepper, signer consensus.Signer, nr int, withTimeout bool) (*Helper, []byte) {
	h := NewHelper(eb, rpcbus, stepper, signer, nr)
	roundUpdate := consensus.MockRoundUpdate(1, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	h.Spawn(hash)
	return h, hash
}
