package reduction

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
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

func (m *mockSigner) SendAuthenticated(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}

// FactoryFunc is a shorthand for the reduction factories to create a Reducer
type FactoryFunc func(*eventbus.EventBus, *rpcbus.RPCBus, key.ConsensusKeys, time.Duration) Reducer

// Helper for reducing test boilerplate
type Helper struct {
	Reducer   Reducer
	Bus       *eventbus.EventBus
	RBus      *rpcbus.RPCBus
	Keys      []key.ConsensusKeys
	P         *user.Provisioners
	signer    consensus.Signer
	PubKeyBLS []byte
	nr        int
	lock      sync.RWMutex
	Handler   *Handler
	stepLock  sync.Mutex
	step      uint8
	State     State
	Round     uint64
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, factory FactoryFunc) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	red := factory(eb, rpcbus, keys[0], 1000*time.Millisecond)
	hlp := &Helper{
		Bus:     eb,
		RBus:    rpcbus,
		Keys:    keys,
		P:       p,
		Reducer: red,
		signer:  &mockSigner{eb},
		nr:      provisioners,
		Handler: NewHandler(keys[0], *p),
		Round:   round,
	}

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

// SendBatch of consensus events to the reducer callback CollectReductionEvent
func (hlp *Helper) SendBatch(hash []byte) {
	batch := hlp.Spawn(hash)
	for _, ev := range batch {
		go hlp.Reducer.Collect(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) []consensus.Event {
	evs := make([]consensus.Event, hlp.nr)
	step := hlp.Step()
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := MockConsensusEvent(hash, round, step, hlp.Keys, vc, i)
		evs[i] = ev

	}
	return evs
}

// Initialize the reducer with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.step = uint8(1)
	hlp.Reducer.Initialize(hlp, hlp.signer, ru)
}
