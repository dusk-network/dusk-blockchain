package secondstep

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
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
	AgreementChan chan bytes.Buffer
	RegenChan     chan bytes.Buffer
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int) *Helper {
	hlp := &Helper{
		Helper:        reduction.NewHelper(eb, rpcbus, provisioners, CreateReducer),
		AgreementChan: make(chan bytes.Buffer, 1),
		RegenChan:     make(chan bytes.Buffer, 1),
	}
	hlp.createResultChan()
	return hlp
}

// CreateResultChan is used by tests (internal and external) to quickly wire the StepVotes resulting from the firststep reduction to a channel to listen to
func (hlp *Helper) createResultChan() {
	agListener := eventbus.NewChanListener(hlp.AgreementChan)
	hlp.Bus.Subscribe(topics.Agreement, agListener)
	regenListener := eventbus.NewChanListener(hlp.RegenChan)
	hlp.Bus.Subscribe(topics.Regeneration, regenListener)
}

// ActivateReduction starts/resumes the secondstep reduction by sending a StepVotes to Reducer.CollectStepVotes
func (hlp *Helper) ActivateReduction(hash []byte, sv *agreement.StepVotes) error {
	buf := new(bytes.Buffer)
	if sv != nil {
		if err := agreement.MarshalStepVotes(buf, sv); err != nil {
			return err
		}
	}
	e := consensus.Event{header.Header{Round: hlp.Round, Step: hlp.Step(), PubKeyBLS: hlp.PubKeyBLS, BlockHash: hash}, *buf}
	hlp.Forward()
	hlp.Reducer.(*Reducer).CollectStepVotes(e)
	return nil
}

// Kickstart creates a Helper and wires up the tests
func Kickstart(nr int) (*Helper, []byte) {
	eb, rpcbus := eventbus.New(), rpcbus.New()
	h := NewHelper(eb, rpcbus, nr)
	roundUpdate := consensus.MockRoundUpdate(1, h.P, nil)
	h.Initialize(roundUpdate)
	hash, _ := crypto.RandEntropy(32)
	return h, hash
}
