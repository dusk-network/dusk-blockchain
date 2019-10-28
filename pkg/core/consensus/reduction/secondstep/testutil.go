package secondstep

import (
	"bytes"
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

func (m *mockSigner) SendAuthenticated(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}

// Helper for reducing test boilerplate
type Helper struct {
	*Factory
	Reducer       reduction.Reducer
	AgreementChan chan bytes.Buffer
	RegenChan     chan bytes.Buffer

	Keys      []key.ConsensusKeys
	P         *user.Provisioners
	signer    consensus.Signer
	PubKeyBLS []byte
	nr        int
	lock      sync.RWMutex
	Handler   *reduction.Handler
	stepLock  sync.Mutex
	step      uint8
	State     State
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	factory := NewFactory(eb, rpcbus, keys[0], 1000*time.Millisecond)
	a := factory.Instantiate()
	red := a.(reduction.Reducer)
	hlp := &Helper{
		Factory:       factory,
		Keys:          keys,
		P:             p,
		Reducer:       red,
		signer:        &mockSigner{eb},
		AgreementChan: make(chan bytes.Buffer, 1),
		RegenChan:     make(chan bytes.Buffer, 1),
		nr:            provisioners,
		Handler:       reduction.NewHandler(keys[0], *p),
	}
	hlp.createResultChan()
	return hlp
}

func (hlp *Helper) Verify(hash []byte, sv *agreement.StepVotes, round uint64, step uint8) error {
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

// CreateResultChan is used by tests (internal and external) to quickly wire the StepVotes resulting from the firststep reduction to a channel to listen to
func (hlp *Helper) createResultChan() {
	agListener := eventbus.NewChanListener(hlp.AgreementChan)
	hlp.Bus.Subscribe(topics.Agreement, agListener)
	regenListener := eventbus.NewChanListener(hlp.RegenChan)
	hlp.Bus.Subscribe(topics.Regeneration, regenListener)
}

// SendBatch of consensus events to the reducer callback Collect
func (hlp *Helper) SendBatch(hash []byte, round uint64, step uint8) {
	batch := hlp.Spawn(hash, round, step)
	for _, ev := range batch {
		go hlp.Reducer.Collect(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte, round uint64, step uint8) []consensus.Event {
	evs := make([]consensus.Event, hlp.nr)
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := reduction.MockConsensusEvent(hash, round, step, hlp.Keys, vc, i)
		evs[i] = ev

	}
	return evs
}

// Initialize the reducer with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.Reducer.Initialize(hlp, hlp.signer, ru)
}

func (hlp *Helper) StartReduction(sv *agreement.StepVotes) error {
	buf := new(bytes.Buffer)
	if sv != nil {
		if err := agreement.MarshalStepVotes(buf, sv); err != nil {
			return err
		}
	}

	hlp.Reducer.(*Reducer).CollectStepVotes(consensus.Event{header.Header{}, *buf})
	return nil
}

func Kickstart(nr int) (*Helper, []byte) {
	eb, rpcbus := eventbus.New(), rpcbus.New()
	h := NewHelper(eb, rpcbus, nr)
	roundUpdate := consensus.MockRoundUpdate(1, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	return h, hash
}
