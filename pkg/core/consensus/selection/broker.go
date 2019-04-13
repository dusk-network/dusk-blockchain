package selection

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// broker is the component that supervises a collection of events
type scoreBroker struct {
	roundUpdateChan  <-chan uint64
	regenerationChan <-chan consensus.AsyncState
	bidListChan      <-chan user.BidList
	filter           *consensus.EventFilter
	selector         *eventSelector
}

// LaunchScoreSelectionComponent creates and launches the component which responsibility is to validate and select the best score among the blind bidders. The component publishes under the topic BestScoreTopic
func LaunchScoreSelectionComponent(eventBroker wire.EventBroker, committee committee.Committee,
	timeout time.Duration) *scoreBroker {
	handler := newScoreHandler()
	broker := newScoreBroker(eventBroker, committee, handler, timeout)
	go broker.Listen()
	return broker
}

func launchScoreFilter(eventBroker wire.EventBroker, committee committee.Committee,
	handler consensus.EventHandler, state consensus.State,
	processor consensus.EventProcessor) *consensus.EventFilter {

	filter := consensus.NewEventFilter(eventBroker, committee, handler, state, processor,
		false)
	go wire.NewTopicListener(eventBroker, filter, string(topics.Score)).Accept()
	return filter
}

// newScoreBroker creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func newScoreBroker(eventBroker wire.EventBroker, committee committee.Committee,
	handler scoreEventHandler, timeOut time.Duration) *scoreBroker {
	//creating the channel whereto notifications about round updates are push onto
	roundChan := consensus.InitRoundUpdate(eventBroker)
	regenerationChan := consensus.InitBlockRegenerationCollector(eventBroker)
	bidListChan := consensus.InitBidListUpdate(eventBroker)
	state := consensus.NewState()
	selector := newEventSelector(eventBroker, handler, timeOut, state)
	filter := launchScoreFilter(eventBroker, committee, handler, state, selector)

	return &scoreBroker{
		filter:           filter,
		roundUpdateChan:  roundChan,
		bidListChan:      bidListChan,
		regenerationChan: regenerationChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *scoreBroker) Listen() {
	for {
		select {
		case round := <-f.roundUpdateChan:
			log.WithFields(log.Fields{
				"process": "selection",
				"round":   round,
			}).Debugln("updating round")

			f.selector.stopSelection()
			f.filter.UpdateRound(round)
			f.filter.FlushQueue()
			go f.selector.startSelection()
		case state := <-f.regenerationChan:
			log.WithFields(log.Fields{
				"process": "selection",
				"round":   state.Round,
				"step":    state.Step,
			}).Debugln("received regeneration message")

			f.selector.RLock()
			if !f.selector.running {
				go f.selector.startSelection()
			}
			f.selector.RUnlock()
		case bidList := <-f.bidListChan:
			f.selector.handler.UpdateBidList(bidList)
		}
	}
}
