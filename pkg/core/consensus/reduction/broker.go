package reduction

import (
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	// Broker is the message broker for the reduction process.
	broker struct {
		filter  *consensus.EventFilter
		reducer *reducer

		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		stepChan        <-chan struct{}
		selectionChan   <-chan *selection.ScoreEvent
	}
)

// Launch creates and wires a broker, initiating the components that
// have to do with Block Reduction
func Launch(eventBroker wire.EventBroker, committee Reducers, keys user.Keys,
	timeout time.Duration, rpcBus *wire.RPCBus) {
	if committee == nil {
		committee = newReductionCommittee(eventBroker, nil)
	}

	handler := newReductionHandler(committee, keys)
	broker := newBroker(eventBroker, handler, timeout, rpcBus)
	go broker.Listen()
}

func launchReductionFilter(eventBroker wire.EventBroker, ctx *context) *consensus.EventFilter {

	filter := consensus.NewEventFilter(ctx.handler, ctx.state, true)
	republisher := consensus.NewRepublisher(eventBroker, topics.Reduction)
	eventBroker.SubscribeCallback(string(topics.Reduction), filter.Collect)
	eventBroker.RegisterPreprocessor(string(topics.Reduction), republisher, &consensus.Validator{})
	return filter
}

// newBroker will return a reduction broker.
func newBroker(eventBroker wire.EventBroker, handler *reductionHandler, timeout time.Duration, rpcBus *wire.RPCBus) *broker {
	scoreChan := initBestScoreUpdate(eventBroker)
	ctx := newCtx(handler, timeout)
	filter := launchReductionFilter(eventBroker, ctx)
	roundChannel := consensus.InitRoundUpdate(eventBroker)

	return &broker{
		roundUpdateChan: roundChannel,
		ctx:             ctx,
		filter:          filter,
		selectionChan:   scoreChan,
		reducer:         newReducer(ctx, eventBroker, filter, rpcBus),
	}
}

// Listen for incoming messages.
func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundUpdateChan:
			log.WithFields(log.Fields{
				"process": "reduction",
				"round":   round,
			}).Debug("Got round update")
			b.reducer.end()
			b.reducer.lock.Lock()
			b.filter.UpdateRound(round)
			b.ctx.timer.ResetTimeOut()
			b.reducer.lock.Unlock()
		case ev := <-b.selectionChan:
			if ev == nil {
				log.WithFields(log.Fields{
					"process": "reduction",
				}).Debug("got empty selection message")
				b.reducer.startReduction(make([]byte, 32))
				b.filter.FlushQueue()
			} else if ev.Round == b.ctx.state.Round() {
				log.WithFields(log.Fields{
					"process": "reduction",
					"hash":    hex.EncodeToString(ev.VoteHash),
				}).Debug("got selection message")
				b.reducer.startReduction(ev.VoteHash)
				b.filter.FlushQueue()
			} else {
				log.WithFields(log.Fields{
					"process":     "reduction",
					"event round": ev.Round,
				}).Debug("got obsolete selection message")
				b.reducer.startReduction(make([]byte, 32))
				b.filter.FlushQueue()
			}
		}
	}
}
