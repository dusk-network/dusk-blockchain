package selection

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// broker is the component that supervises a collection of events
type scoreBroker struct {
	roundUpdateChan  <-chan uint64
	regenerationChan <-chan consensus.AsyncState
	bidChan          <-chan user.Bid
	filter           *filter
	selector         *eventSelector
	handler          ScoreEventHandler
}

// Launch creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func Launch(eventBroker wire.EventBroker, handler ScoreEventHandler, timeout time.Duration) {
	if handler == nil {
		handler = newScoreHandler()
	}
	broker := newScoreBroker(eventBroker, handler, timeout)
	go broker.Listen()
}

func launchScoreFilter(eventBroker wire.EventBroker, handler ScoreEventHandler,
	state consensus.State, selector *eventSelector) *filter {

	filter := newFilter(handler, state, selector)
	listener := wire.NewTopicListener(eventBroker, filter, string(topics.Score))
	go listener.Accept()
	return filter
}

// newScoreBroker creates a Broker component which responsibility is to listen to the
// eventbus and supervise Collector operations
func newScoreBroker(eventBroker wire.EventBroker, handler ScoreEventHandler, timeOut time.Duration) *scoreBroker {
	state := consensus.NewState()
	selector := newEventSelector(eventBroker, handler, timeOut, state)
	filter := launchScoreFilter(eventBroker, handler, state, selector)

	return &scoreBroker{
		filter:           filter,
		roundUpdateChan:  consensus.InitRoundUpdate(eventBroker),
		bidChan:          consensus.InitBidListUpdate(eventBroker),
		regenerationChan: consensus.InitBlockRegenerationCollector(eventBroker),
		selector:         selector,
		handler:          handler,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *scoreBroker) Listen() {
	for {
		select {
		case round := <-f.roundUpdateChan:
			f.onRoundUpdate(round)
		case state := <-f.regenerationChan:
			f.onRegeneration(state)
		case bid := <-f.bidChan:
			f.selector.handler.UpdateBidList(bid)
		}
	}
}

func (f *scoreBroker) onRoundUpdate(round uint64) {
	log.WithFields(log.Fields{
		"process": "selection",
		"round":   round,
	}).Debugln("updating round")

	f.filter.UpdateRound(round)
	f.selector.stopSelection()
	f.handler.ResetThreshold()
	f.handler.RemoveExpiredBids(round)
	f.filter.FlushQueue()
	f.selector.startSelection()
	f.selector.timer.ResetTimeOut()
}

func (f *scoreBroker) onRegeneration(state consensus.AsyncState) {
	log.WithFields(log.Fields{
		"process": "selection",
		"round":   state.Round,
		"step":    state.Step,
	}).Debugln("received regeneration message")
	if state.Round == f.selector.state.Round() {
		f.handler.LowerThreshold()
		f.selector.timer.IncreaseTimeOut()
		if !f.selector.isRunning() {
			f.selector.startSelection()
			return
		}
	}
}
