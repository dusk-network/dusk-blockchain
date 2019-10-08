package reduction

import (
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
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
		filter  *consensus.EventFilter
		Reducer *reducer

		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan consensus.RoundUpdate
		selectionChan   <-chan *selection.ScoreEvent
	}
)

func initBroker(eventBroker eventbus.Broker, keys user.Keys, timeout time.Duration, rpcBus *rpcbus.RPCBus) *broker {
	handler := newReductionHandler(keys)
	return newBroker(eventBroker, handler, timeout, rpcBus)
}

// Launch creates and wires a broker, initiating the components that
// have to do with Block Reduction
func Launch(eventBroker eventbus.Broker, keys user.Keys, timeout time.Duration, rpcBus *rpcbus.RPCBus) {
	broker := initBroker(eventBroker, keys, timeout, rpcBus)
	go broker.Listen()
}

func launchReductionFilter(eventBroker eventbus.Broker, ctx *context) *consensus.EventFilter {

	filter := consensus.NewEventFilter(ctx.handler, ctx.state, true)
	republisher := consensus.NewRepublisher(eventBroker, topics.Reduction)
	cbListener := eventbus.NewCallbackListener(filter.Collect)
	eventBroker.Subscribe(topics.Reduction, cbListener)
	eventBroker.Register(topics.Reduction, republisher, &consensus.Validator{})
	return filter
}

// newBroker will return a reduction broker.
func newBroker(eventBroker eventbus.Broker, handler *reductionHandler, timeout time.Duration, rpcBus *rpcbus.RPCBus) *broker {
	scoreChan := initBestScoreUpdate(eventBroker)
	ctx := newCtx(handler, timeout)
	filter := launchReductionFilter(eventBroker, ctx)
	roundChannel := consensus.InitRoundUpdate(eventBroker)

	return &broker{
		roundUpdateChan: roundChannel,
		ctx:             ctx,
		filter:          filter,
		selectionChan:   scoreChan,
		Reducer:         newReducer(ctx, eventBroker, filter, rpcBus),
	}
}

func (b *broker) propagateRound(roundUpdate consensus.RoundUpdate) {
	log.WithFields(log.Fields{
		"process": "reduction",
		"round":   roundUpdate.Round,
	}).Debug("Got round update")
	b.Reducer.end()
	b.Reducer.lock.Lock()
	b.ctx.handler.UpdateProvisioners(roundUpdate.P)
	b.filter.UpdateRound(roundUpdate.Round)
	b.ctx.timer.ResetTimeOut()
	b.Reducer.lock.Unlock()
}

func (b *broker) propagateScore(ev *selection.ScoreEvent) {
	if ev == nil {
		log.WithFields(log.Fields{
			"process": "reduction",
		}).Debug("got empty selection message")
		b.Reducer.startReduction(make([]byte, 32))
	} else if ev.Round == b.ctx.state.Round() {
		log.WithFields(log.Fields{
			"process": "reduction",
			"hash":    hex.EncodeToString(ev.VoteHash),
		}).Debug("got selection message")
		b.Reducer.startReduction(ev.VoteHash)
	} else {
		log.WithFields(log.Fields{
			"process":     "reduction",
			"event round": ev.Round,
		}).Debug("got obsolete selection message")
		b.Reducer.startReduction(make([]byte, 32))
	}

	b.filter.FlushQueue()
}

// Listen for incoming messages.
func (b *broker) Listen() {
	for {
		select {
		case roundUpdate := <-b.roundUpdateChan:
			b.propagateRound(roundUpdate)

		case ev := <-b.selectionChan:
			b.propagateScore(ev)
		}
	}
}
