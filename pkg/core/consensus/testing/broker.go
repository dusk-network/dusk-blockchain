package testing

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// Provides GetLastCommitte, GetLastCertificate, VerifyCandidateBlock and GetCandidate,
// Updates topics.Candidate
// rpcbus-friendly component
type mockSafeRegistryBroker struct {
	getLastCertificateChan <-chan rpcbus.Request
	getLastCommitteeChan   <-chan rpcbus.Request
	getLastBlock           <-chan rpcbus.Request

	candidateChan chan message.Message

	reg *mockSafeRegistry
}

func newMockSafeRegistryBroker(e consensus.Emitter, reg *mockSafeRegistry) (*mockSafeRegistryBroker, error) {

	// Subscriptions

	candidateChan := make(chan message.Message, 1)
	chanListener := eventbus.NewChanListener(candidateChan)
	e.EventBus.Subscribe(topics.Candidate, chanListener)

	// register rpcbus
	// set up rpcbus channels
	getLastCertificateChan := make(chan rpcbus.Request, 1)
	getLastCommitteeChan := make(chan rpcbus.Request, 1)
	getLastBlock := make(chan rpcbus.Request, 1)

	if err := e.RPCBus.Register(topics.GetLastCertificate, getLastCertificateChan); err != nil {
		return nil, err
	}

	if err := e.RPCBus.Register(topics.GetLastCommittee, getLastCommitteeChan); err != nil {
		return nil, err
	}

	if err := e.RPCBus.Register(topics.GetLastBlock, getLastBlock); err != nil {
		return nil, err
	}

	return &mockSafeRegistryBroker{
		candidateChan:          candidateChan,
		reg:                    reg,
		getLastCertificateChan: getLastCertificateChan,
		getLastCommitteeChan:   getLastCommitteeChan,
		getLastBlock:           getLastBlock,
	}, nil
}

// mockConsensusRegistryProvider provides async read-only access to ConsensusRegistry needed by P2P layer
func (c *mockSafeRegistryBroker) loop(assert *assert.Assertions) {
	for {
		select {
		// HighestSeen
		case r := <-c.getLastBlock:
			c.provideLastBlock(r)
		case r := <-c.getLastCertificateChan:
			c.provideLastCertificate(r)
		case r := <-c.getLastCommitteeChan:
			c.provideLastCommittee(r)
		case m := <-c.candidateChan:
			cm := m.Payload().(message.Candidate)
			// TODO: ValidateCandidate
			//if err := ValidateCandidate(cm); err != nil {
			//	diagnostics.LogError("error in validating the candidate", err)
			//	return
			//}
			c.reg.AddCandidate(cm)
		}
	}
}

func (c *mockSafeRegistryBroker) provideLastCertificate(r rpcbus.Request) {
	cert := c.reg.GetLastCertificate()
	if cert == nil {
		r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("no last certificate present"))
		return
	}

	buf := new(bytes.Buffer)
	err := message.MarshalCertificate(buf, cert)
	r.RespChan <- rpcbus.NewResponse(*buf, err)
}

func (c *mockSafeRegistryBroker) provideLastCommittee(r rpcbus.Request) {
	committee := c.reg.GetLastCommittee()
	if committee == nil {
		r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("no last committee present"))
		return
	}

	r.RespChan <- rpcbus.NewResponse(committee, nil)
}

func (c *mockSafeRegistryBroker) provideLastBlock(r rpcbus.Request) {
	b := c.reg.GetChainTip()
	r.RespChan <- rpcbus.NewResponse(b, nil)
}
