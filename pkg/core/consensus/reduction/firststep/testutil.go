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

const round = uint64(1)

// State indicates the status of the EventPlayer
type State uint8

const (
	// PAUSED player
	PAUSED State = iota
	// RUNNING player
	RUNNING
)

type mockSigner struct {
	bus *eventbus.EventBus
}

func (m *mockSigner) Sign([]byte, []byte) ([]byte, error) {
	return make([]byte, 33), nil
}
func (m *mockSigner) SendAuthenticated(topics.Topic, []byte, *bytes.Buffer) error { return nil }
func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}

// Helper for reducing test boilerplate
type Helper struct {
	*Factory
	Keys    []key.ConsensusKeys
	P       *user.Provisioners
	Reducer *Reducer
	signer  consensus.Signer

	StepVotesChan chan bytes.Buffer
	nr            int

	lock               sync.RWMutex
	failOnVerification bool
	Handler            *reduction.Handler

	stepLock sync.Mutex
	step     uint8
	State    State
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	factory := NewFactory(eb, rpcbus, keys[0], 1000*time.Millisecond)
	a := factory.Instantiate()
	red := a.(*Reducer)
	hlp := &Helper{
		Factory:            factory,
		Keys:               keys,
		P:                  p,
		Reducer:            red,
		signer:             &mockSigner{eb},
		StepVotesChan:      make(chan bytes.Buffer, 1),
		nr:                 provisioners,
		failOnVerification: false,
		Handler:            reduction.NewHandler(keys[0], *p),
	}
	go hlp.verifyCandidateBlock()
	hlp.createResultChan()
	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper
func (hlp *Helper) Verify(hash []byte, sv *agreement.StepVotes, step uint8) error {
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	sub := vc.Intersect(sv.BitSet)
	apk, err := agreement.ReconstructApk(sub)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, hash, apk, sv.Signature)
}

// Forward upticks the step
func (hlp *Helper) Forward() {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.step++
}

// Step guards the step with a lock
func (hlp *Helper) Step() uint8 {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	return hlp.step
}

// Pause as specified by the EventPlayer interface
func (hlp *Helper) Pause(id uint32) {
	hlp.State = PAUSED
}

// Resume as specified by the EventPlayer interface
func (hlp *Helper) Resume(id uint32) {
	hlp.State = RUNNING
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
	step := hlp.Step()
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := reduction.MockConsensusEvent(hash, round, step, hlp.Keys, vc, i)
		evs[i] = ev

	}
	return evs
}

// Initialize the reducer with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.step = uint8(1)
	hlp.Reducer.Initialize(hlp, hlp.signer, ru)
}

// StartReduction sends the reducer a BestScore event to trigger a EvenPlayer.Resume
func (hlp *Helper) StartReduction(hash []byte) {
	hdr := header.Header{BlockHash: hash}
	hlp.Reducer.CollectBestScore(consensus.Event{hdr, bytes.Buffer{}})
}

// Kickstart a Helper without sending any reduction event
func Kickstart(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, nr int) (*Helper, []byte) {
	h := NewHelper(eb, rpcbus, nr)
	roundUpdate := consensus.MockRoundUpdate(round, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	h.StartReduction(hash)
	return h, hash
}

// ProduceFirstStepVotes encapsulates the process of creating and forwarding Reduction events
func ProduceFirstStepVotes(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, nr int) (*Helper, []byte) {
	hlp, hash := Kickstart(eb, rpcbus, nr)
	hlp.SendBatch(hash)
	return hlp, hash
}
