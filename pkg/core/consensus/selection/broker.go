package selection

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// broker is the component that supervises a collection of events
type scoreBroker struct {
	roundUpdateChan  <-chan consensus.RoundUpdate
	regenerationChan <-chan consensus.AsyncState
	filter           *filter
	selector         *eventSelector
	handler          ScoreEventHandler
}

// Launch creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func Launch(eventBroker eventbus.Broker, handler ScoreEventHandler, timeout time.Duration) {
	if handler == nil {
		handler = newScoreHandler()
	}
	broker := newScoreBroker(eventBroker, handler, timeout)
	go broker.Listen()
}

func launchScoreFilter(eventBroker eventbus.Broker, handler ScoreEventHandler,
	state consensus.State, selector *eventSelector) *filter {

	filter := newFilter(handler, state, selector)
	eventbus.NewTopicListener(eventBroker, filter, string(topics.Score), eventbus.ChannelType)
	return filter
}

// newScoreBroker creates a Broker component which responsibility is to listen to the
// eventbus and supervise Collector operations
func newScoreBroker(eventBroker eventbus.Broker, handler ScoreEventHandler, timeOut time.Duration) *scoreBroker {
	state := consensus.NewState()
	selector := newEventSelector(eventBroker, handler, timeOut, state)
	filter := launchScoreFilter(eventBroker, handler, state, selector)

	return &scoreBroker{
		filter:           filter,
		roundUpdateChan:  consensus.InitRoundUpdate(eventBroker),
		regenerationChan: consensus.InitBlockRegenerationCollector(eventBroker),
		selector:         selector,
		handler:          handler,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *scoreBroker) Listen() {
	for {
		select {
		case roundUpdate := <-f.roundUpdateChan:
			f.onRoundUpdate(roundUpdate)
		case state := <-f.regenerationChan:
			f.onRegeneration(state)
		}
	}
}

func (f *scoreBroker) onRoundUpdate(roundUpdate consensus.RoundUpdate) {
	log.WithFields(log.Fields{
		"process": "selection",
		"round":   roundUpdate.Round,
	}).Debugln("updating round")

	f.filter.UpdateRound(roundUpdate.Round)
	f.selector.stopSelection()
	f.handler.ResetThreshold()
	f.handler.UpdateBidList(roundUpdate.BidList)
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
