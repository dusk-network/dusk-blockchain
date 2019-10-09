package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

type (
	// Broker is the message broker for the reduction process.
	broker struct {
		Reducer *reducer
		handler *reductionHandler
	}
)

func NewComponent(publisher eventbus.Publisher, keys user.Keys, timeout time.Duration, rpcBus *rpcbus.RPCBus, p user.Provisioners, requestStepUpdate func()) *broker {
	handler := newReductionHandler(keys)
	return newBroker(eventBroker, handler, timeout, rpcBus)
}

// newBroker will return a reduction broker.
func newBroker(publisher eventbus.Publisher, handler *reductionHandler, timeOut time.Duration, rpcBus *rpcbus.RPCBus, requestStepUpdate func()) *broker {
	b := &broker{
		handler: handler,
		Reducer: newReducer(ctx, eventBroker, timeOut, filter, rpcBus, requestStepUpdate),
	}
	return b
}

func (b *broker) Initialize() ([]consensus.Subscriber, consensus.EventHandler) {
	reductionSubscriber := consensus.Subscriber{
		topic:    topics.Reduction,
		listener: consensus.NewCallbackListener(b.CollectReductionEvent),
	}

	scoreSubscriber := consensus.Subscriber{
		topic:    topics.BestScore,
		listener: consensus.NewCallbackListener(b.CollectBestScore),
	}

	return []Subscriber{reductionSubscriber, scoreSubscriber}, b.handler
}

func (b *broker) Initialize() ([]Subscriber, EventHandler) {
	b.handler.UpdateProvisioners(roundUpdate.P)
}

func (b *broker) Finalize() {
	log.WithFields(log.Fields{
		"process": "reduction",
		"round":   roundUpdate.Round,
	}).Debug("Got round update")
	b.Reducer.end()
	b.timer.Close()
}

func (b *broker) CollectReductionEvent(m bytes.Buffer, hdr header.Header) error {
	ev := New()
	if err := reduction.Unmarshal(&m, ev); err != nil {
		return err
	}

	if err := b.handler.VerifySignature(hdr, ev.SignedHash); err != nil {
		return err
	}

	return nil
}

func (b *broker) SetStep(step uint8) {
	b.reducer.aggregator.Stop()
	b.reducer.aggregator = newAggregator()
}

func (b *broker) CollectBestScore(r bytes.Buffer, hdr *header.Header) error {
	ev := &selection.ScoreEvent{}
	if err := selection.UnmarshalScoreEvent(&r, ev); err != nil {
		return err
	}

	if len(ev.VoteHash) == 32 {
		b.propagateScore(ev)
		return nil
	}

	b.propagateScore(nil)
	return nil
}

func (b *broker) propagateScore(ev *selection.ScoreEvent) {
	if ev == nil {
		log.WithFields(log.Fields{
			"process": "reduction",
		}).Debug("got empty selection message")
		b.Reducer.startReduction(emptyHash[:])
		return
	}

	log.WithFields(log.Fields{
		"process": "reduction",
		"hash":    hex.EncodeToString(ev.VoteHash),
	}).Debug("got selection message")
	b.Reducer.startReduction(ev.VoteHash)
}
