package responding

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// RoundResultBroker holds RPCBus and responseChan
type RoundResultBroker struct {
	rpcBus       *rpcbus.RPCBus
	responseChan chan<- *bytes.Buffer
}

// NewRoundResultBroker will instantiate new RoundResultBroker
func NewRoundResultBroker(rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer) *RoundResultBroker {
	return &RoundResultBroker{rpcBus, responseChan}
}

// ProvideRoundResult will call the rpc endpoint for round results and prepend it to a topic
func (r *RoundResultBroker) ProvideRoundResult(m *bytes.Buffer) error {
	resp, err := r.rpcBus.Call(topics.GetRoundResults, rpcbus.Request{*m, make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		return err
	}
	roundResultBuf := resp.(bytes.Buffer)

	if err := topics.Prepend(&roundResultBuf, topics.RoundResults); err != nil {
		return err
	}

	r.responseChan <- &roundResultBuf
	return nil
}
