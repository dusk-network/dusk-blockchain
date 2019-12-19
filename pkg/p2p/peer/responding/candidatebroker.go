package responding

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

type CandidateBroker struct {
	rpcBus       *rpcbus.RPCBus
	responseChan chan<- *bytes.Buffer
}

func NewCandidateBroker(rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer) *CandidateBroker {
	return &CandidateBroker{rpcBus, responseChan}
}

func (c *CandidateBroker) ProvideCandidate(m *bytes.Buffer) error {
	candidateBytes, err := c.rpcBus.Call(rpcbus.GetCandidate, rpcbus.Request{*m, make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		return err
	}

	if err := topics.Prepend(&candidateBytes, topics.Candidate); err != nil {
		return err
	}

	c.responseChan <- &candidateBytes
	return nil
}
