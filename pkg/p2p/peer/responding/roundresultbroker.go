package responding

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

type RoundResultBroker struct {
	rpcBus       *rpcbus.RPCBus
	responseChan chan<- *bytes.Buffer
}

func NewRoundResultBroker(rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer) *RoundResultBroker {
	return &RoundResultBroker{rpcBus, responseChan}
}

func (r *RoundResultBroker) ProvideRoundResult(m *bytes.Buffer) error {
	roundResultBuf, err := r.rpcBus.Call(rpcbus.GetRoundResults, rpcbus.Request{*m, make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		return err
	}

	if err := topics.Prepend(&roundResultBuf, topics.RoundResults); err != nil {
		return err
	}

	r.responseChan <- &roundResultBuf
	return nil
}
