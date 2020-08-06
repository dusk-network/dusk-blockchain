package responding

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// CandidateBroker holds instances to RPCBus and responseChan
type CandidateBroker struct {
	rpcBus       *rpcbus.RPCBus
	responseChan chan<- *bytes.Buffer
}

// NewCandidateBroker will create new CandidateBroker
func NewCandidateBroker(rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer) *CandidateBroker {
	return &CandidateBroker{rpcBus, responseChan}
}

// ProvideCandidate for a given (m *bytes.Buffer)
func (c *CandidateBroker) ProvideCandidate(m *bytes.Buffer) error {
	//FIXME: Add option to configure rpcBus timeout #614
	resp, err := c.rpcBus.Call(topics.GetCandidate, rpcbus.NewRequest(*m), 5*time.Second)
	if err != nil {
		lg.
			WithError(err).
			Error("timeout ProvideCandidate topics.GetCandidate")
		return err
	}
	cm := resp.(message.Candidate)

	candidateBytes := new(bytes.Buffer)
	if err := message.MarshalCandidate(candidateBytes, cm); err != nil {
		return err
	}

	if err := topics.Prepend(candidateBytes, topics.Candidate); err != nil {
		return err
	}

	c.responseChan <- candidateBytes
	return nil
}
