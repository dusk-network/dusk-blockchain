package secondstep

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
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
	AgreementChan chan message.Message
	RestartChan   chan message.Message
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, timeOut time.Duration) *Helper {
	hlp := &Helper{
		Helper:        reduction.NewHelper(eb, rpcbus, provisioners, CreateReducer, timeOut),
		AgreementChan: make(chan message.Message, 1),
		RestartChan:   make(chan message.Message, 1),
	}
	hlp.createResultChan()
	return hlp
}

// CreateResultChan is used by tests (internal and external) to quickly wire the StepVotes resulting from the firststep reduction to a channel to listen to
func (hlp *Helper) createResultChan() {
	agListener := eventbus.NewChanListener(hlp.AgreementChan)
	hlp.Bus.Subscribe(topics.Agreement, agListener)
	restartListener := eventbus.NewChanListener(hlp.RestartChan)
	hlp.Bus.Subscribe(topics.Restart, restartListener)
}

// ActivateReduction starts/resumes the secondstep reduction by sending a StepVotes to Reducer.CollectStepVotes
func (hlp *Helper) ActivateReduction(hash []byte, sv *message.StepVotes) error {
	// e := consensus.Event{header.Header{Round: hlp.Round, Step: hlp.Step(), PubKeyBLS: hlp.PubKeyBLS, BlockHash: hash}, *buf}
	msg := message.NewStepVotesMsg(hlp.Round, hash, *sv)
	hlp.Reducer.(*Reducer).CollectStepVotes(msg)
	return nil
}

// Kickstart creates a Helper and wires up the tests
func Kickstart(nr int, timeOut time.Duration) (*Helper, []byte) {
	eb, rpcbus := eventbus.New(), rpcbus.New()
	h := NewHelper(eb, rpcbus, nr, timeOut)
	roundUpdate := consensus.MockRoundUpdate(1, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	return h, hash
}
