package testing

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Provides GetLastCommitte, GetLastCertificate, VerifyCandidateBlock and GetCandidate,
// Updates topics.Candidate
// rpcbus-friendly component
//nolint:unused
type mockSafeRegistryBroker struct {
	getLastCertificateChan <-chan rpcbus.Request
	getLastCommitteeChan   <-chan rpcbus.Request
	getLastBlock           <-chan rpcbus.Request
	getCandidate           <-chan rpcbus.Request
	getMempoool            <-chan rpcbus.Request

	candidateChan chan message.Message

	reg *mockSafeRegistry
}

//nolint:unused
func newMockSafeRegistryBroker(e consensus.Emitter, reg *mockSafeRegistry) (*mockSafeRegistryBroker, error) {

	// Subscriptions

	candidateChan := make(chan message.Message, 10)
	chanListener := eventbus.NewChanListener(candidateChan)
	e.EventBus.Subscribe(topics.Candidate, chanListener)

	// register rpcbus
	// set up rpcbus channels
	getLastCertificateChan := make(chan rpcbus.Request, 1)
	getLastCommitteeChan := make(chan rpcbus.Request, 1)
	getLastBlock := make(chan rpcbus.Request, 1)
	getCandidate := make(chan rpcbus.Request, 1)

	if err := e.RPCBus.Register(topics.GetLastCertificate, getLastCertificateChan); err != nil {
		return nil, err
	}

	if err := e.RPCBus.Register(topics.GetLastCommittee, getLastCommitteeChan); err != nil {
		return nil, err
	}

	if err := e.RPCBus.Register(topics.GetLastBlock, getLastBlock); err != nil {
		return nil, err
	}

	if err := e.RPCBus.Register(topics.GetCandidate, getCandidate); err != nil {
		return nil, err
	}

	getMempoool := make(chan rpcbus.Request, 10)
	if err := e.RPCBus.Register(topics.GetMempoolTxsBySize, getMempoool); err != nil {
		panic(err)
	}

	return &mockSafeRegistryBroker{
		candidateChan:          candidateChan,
		reg:                    reg,
		getLastCertificateChan: getLastCertificateChan,
		getLastCommitteeChan:   getLastCommitteeChan,
		getLastBlock:           getLastBlock,
		getMempoool:            getMempoool,
		getCandidate:           getCandidate,
	}, nil
}

/// Replace Candidate Broker
// Merge CandidateBroker and LastBlockProvider routine
// mockConsensusRegistryProvider provides async read-only access to ConsensusRegistry needed by P2P layer
func (c *mockSafeRegistryBroker) loop(pctx context.Context) {
	for {
		select {
		case r := <-c.getMempoool:
			r.RespChan <- rpcbus.NewResponse([]transactions.ContractCall{}, nil)
		case r := <-c.getCandidate:
			c.provideCandidate(r)
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
		case <-pctx.Done():
			return
		}
	}
}

func (c *mockSafeRegistryBroker) provideCandidate(r rpcbus.Request) {
	// Read params from request
	params := r.Params.(bytes.Buffer)

	// Mandatory param
	hash := make([]byte, 32)
	if err := encoding.Read256(&params, hash); err != nil {
		r.RespChan <- rpcbus.Response{Resp: nil, Err: err}
		return
	}

	// Enforce fetching from peers if local cache does not have this Candidate
	// Optional param
	var fetchFromPeers bool
	_ = encoding.ReadBool(&params, &fetchFromPeers)

	cm, err := c.reg.GetCandidateByHash(hash)
	r.RespChan <- rpcbus.Response{Resp: cm, Err: err}
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
