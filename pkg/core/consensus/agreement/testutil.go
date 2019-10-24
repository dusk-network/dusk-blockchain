package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// Helper is a struct that facilitates sending semi-real Events with minimum effort
type Helper struct {
	Bus             *eventbus.EventBus
	P               *user.Provisioners
	Keys            []user.Keys
	Aggro           *agreement
	WinningHashChan chan bytes.Buffer
	nr              int
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, provisioners int) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	factory := NewFactory(eb, keys[0])
	a := factory.Instantiate()
	aggro := a.(*agreement)
	hlp := &Helper{eb, p, keys, aggro, make(chan bytes.Buffer, 1), provisioners}
	hlp.createResultChan()
	return hlp
}

// CreateResultChan is used by tests (internal and external) to quickly wire the agreement results to a channel to listen to
func (hlp *Helper) createResultChan() {
	chanListener := eventbus.NewChanListener(hlp.WinningHashChan)
	hlp.Bus.Subscribe(topics.WinningBlockHash, chanListener)
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) {
	vc := hlp.P.CreateVotingCommittee(1, 1, hlp.nr)
	for i := 0; i < hlp.nr; i++ {
		ev := MockConsensusEvent(hash, 1, 1, hlp.Keys, vc, i)
		go hlp.Aggro.CollectAgreementEvent(ev)
	}
}

// Initialize the Agreement with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.Aggro.Initialize(nil, nil, ru)
}

// ProduceWinningHash is used to produce enough valid Events to reach Quorum and trigger sending a winning hash to the channel
func ProduceWinningHash(eb *eventbus.EventBus, nr int) (*Helper, []byte) {
	hlp := NewHelper(eb, nr)
	roundUpdate := consensus.MockRoundUpdate(1, hlp.P, nil)
	hlp.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	hlp.Spawn(hash)
	return hlp, hash
}
