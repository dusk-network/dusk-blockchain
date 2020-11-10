package chain2

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// SafeRegistryBroker provides GetLastCommitte, GetLastCertificate, VerifyCandidateBlock and GetCandidate,
type SafeRegistryBroker struct {
	getLastCertificateChan <-chan rpcbus.Request
	getLastCommitteeChan   <-chan rpcbus.Request
	getLastBlock           <-chan rpcbus.Request

	reg *SafeRegistry
}

func newSafeRegistryBroker(r *rpcbus.RPCBus, reg *SafeRegistry) (*SafeRegistryBroker, error) {

	// register rpcbus
	// set up rpcbus channels
	getLastCertificateChan := make(chan rpcbus.Request, 1)
	getLastCommitteeChan := make(chan rpcbus.Request, 1)
	getLastBlock := make(chan rpcbus.Request, 1)

	if err := r.Register(topics.GetLastCertificate, getLastCertificateChan); err != nil {
		return nil, err
	}

	if err := r.Register(topics.GetLastCommittee, getLastCommitteeChan); err != nil {
		return nil, err
	}

	if err := r.Register(topics.GetLastBlock, getLastBlock); err != nil {
		return nil, err
	}

	return &SafeRegistryBroker{
		reg:                    reg,
		getLastCertificateChan: getLastCertificateChan,
		getLastCommitteeChan:   getLastCommitteeChan,
		getLastBlock:           getLastBlock,
	}, nil
}

func (c *SafeRegistryBroker) loop(pctx context.Context) {
	for {
		select {
		case r := <-c.getLastBlock:
			c.provideLastBlock(r)
		case r := <-c.getLastCertificateChan:
			c.provideLastCertificate(r)
		case r := <-c.getLastCommitteeChan:
			c.provideLastCommittee(r)
		case <-pctx.Done():
			return
		}
	}
}

func (c *SafeRegistryBroker) provideLastCertificate(r rpcbus.Request) {
	cert := c.reg.GetLastCertificate()
	if cert == nil {
		r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("no last certificate present"))
		return
	}

	buf := new(bytes.Buffer)
	err := message.MarshalCertificate(buf, cert)
	r.RespChan <- rpcbus.NewResponse(*buf, err)
}

func (c *SafeRegistryBroker) provideLastCommittee(r rpcbus.Request) {
	committee := c.reg.GetLastCommittee()
	if committee == nil {
		r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("no last committee present"))
		return
	}

	r.RespChan <- rpcbus.NewResponse(committee, nil)
}

func (c *SafeRegistryBroker) provideLastBlock(r rpcbus.Request) {
	b := c.reg.GetChainTip()
	r.RespChan <- rpcbus.NewResponse(b, nil)
}
