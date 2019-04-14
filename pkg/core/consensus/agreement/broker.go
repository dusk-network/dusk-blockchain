package agreement

import (
	"bytes"
	"encoding/binary"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type broker struct {
	publisher   wire.EventPublisher
	state       consensus.State
	filter      *consensus.EventFilter
	accumulator *consensus.Accumulator
}

// LaunchAgreement is a helper to minimize the wiring of TopicListeners,
// collector and channels. The agreement component notarizes the new blocks after having collected a quorum of votes
func LaunchAgreement(eventBus *wire.EventBus, committee committee.Committee,
	currentRound uint64) *broker {
	broker := newBroker(eventBus, committee)
	broker.updateRound(currentRound)
	go broker.Listen()
	return broker
}

func launchFilter(eventBroker wire.EventBroker, committee committee.Committee,
	handler consensus.EventHandler, state consensus.State,
	accumulator *consensus.Accumulator) *consensus.EventFilter {
	filter := consensus.NewEventFilter(committee, handler, state, accumulator, false)
	republisher := consensus.NewRepublisher(eventBroker, topics.Agreement)
	listener := wire.NewTopicListener(eventBroker, filter, string(topics.Agreement))
	go listener.Accept(republisher, &consensus.Validator{})
	return filter
}

func newBroker(eventBroker wire.EventBroker, committee committee.Committee) *broker {
	handler := newHandler(committee)
	accumulator := consensus.NewAccumulator(handler)
	state := consensus.NewState()
	filter := launchFilter(eventBroker, committee, handler,
		state, accumulator)
	return &broker{
		publisher:   eventBroker,
		state:       state,
		filter:      filter,
		accumulator: accumulator,
	}
}

// Listen for results coming from the accumulator
func (b *broker) Listen() {
	for {
		<-b.accumulator.CollectedVotesChan
		b.updateRound(b.state.Round() + 1)
	}
}

func (b *broker) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "agreement",
		"round":   round,
	}).Debugln("updating round")
	b.filter.UpdateRound(round)
	b.publishRoundUpdate(round)
	b.accumulator.Clear()
	// TODO: should consume entire round messages
	b.filter.FlushQueue()
}

func (b *broker) publishRoundUpdate(round uint64) {
	// Marshalling the round update
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, round)
	buf := bytes.NewBuffer(bs)
	// publishing to the EventBus
	b.publisher.Publish(msg.RoundUpdateTopic, buf)
}
