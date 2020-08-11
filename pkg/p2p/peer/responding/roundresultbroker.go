package responding

import (
	"bytes"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var lg = log.WithField("process", "RoundResultBroker")

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
	timeoutGetRoundResults := time.Duration(config.Get().General.TimeoutGetRoundResults) * time.Second
	resp, err := r.rpcBus.Call(topics.GetRoundResults, rpcbus.NewRequest(*m), timeoutGetRoundResults)
	if err != nil {
		lg.
			WithError(err).
			Error("timeout ProvideCandidate topics.GetRoundResults")
		return err
	}
	roundResultBuf := resp.(bytes.Buffer)

	if err := topics.Prepend(&roundResultBuf, topics.RoundResults); err != nil {
		return err
	}

	r.responseChan <- &roundResultBuf
	return nil
}
