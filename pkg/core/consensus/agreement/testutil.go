package agreement

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

// Helper is a struct that facilitates sending semi-real Events with minimum effort
type Helper struct {
	Bus             *eventbus.EventBus
	P               *user.Provisioners
	Keys            []key.ConsensusKeys
	Aggro           *agreement
	CertificateChan chan message.Message
	nr              int
}

func WireAgreement(nrProvisioners int) (*consensus.Coordinator, *Helper) {
	eb := eventbus.New()
	h := NewHelper(eb, nrProvisioners)
	factory := NewFactory(eb, h.Keys[0])
	coordinator := consensus.Start(eb, h.Keys[0], factory)
	// starting up the coordinator
	ru := consensus.MockRoundUpdate(1, h.P, nil)
	msg := message.New(topics.RoundUpdate, ru)
	if err := coordinator.CollectRoundUpdate(msg); err != nil {
		panic(err)
	}
	// Play to step 3, as agreements can only be made on step 3 or later
	// This prevents the mocked events from getting queued
	coordinator.Play(h.Aggro.ID())
	coordinator.Play(h.Aggro.ID())
	return coordinator, h
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, provisioners int) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	factory := NewFactory(eb, keys[0])
	a := factory.Instantiate()
	aggro := a.(*agreement)
	hlp := &Helper{eb, p, keys, aggro, make(chan message.Message, 1), provisioners}
	hlp.createResultChan()
	return hlp
}

// CreateResultChan is used by tests (internal and external) to quickly wire the agreement results to a channel to listen to
func (hlp *Helper) createResultChan() {
	chanListener := eventbus.NewChanListener(hlp.CertificateChan)
	hlp.Bus.Subscribe(topics.Certificate, chanListener)
}

// SendBatch let agreement collect  additional batches of consensus events
func (hlp *Helper) SendBatch(hash []byte) {
	batch := hlp.Spawn(hash)
	for _, ev := range batch {
		go hlp.Aggro.CollectAgreementEvent(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) []message.Agreement {
	evs := make([]message.Agreement, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := message.MockAgreement(hash, 1, 3, hlp.Keys, hlp.P, i)
		evs[i] = ev
	}

	return evs
}

// Initialize the Agreement with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.Aggro.Initialize(consensus.NewSimplePlayer(), nil, ru)
}

func LaunchHelper(eb *eventbus.EventBus, nr int) (*Helper, []byte) {
	hlp := NewHelper(eb, nr)
	roundUpdate := consensus.MockRoundUpdate(1, hlp.P, nil)
	hlp.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	return hlp, hash
}

// ProduceWinningHash is used to produce enough valid Events to reach Quorum and trigger sending a winning hash to the channel
func ProduceWinningHash(eb *eventbus.EventBus, nr int) (*Helper, []byte) {
	hlp, hash := LaunchHelper(eb, nr)
	hlp.SendBatch(hash)
	return hlp, hash
}
