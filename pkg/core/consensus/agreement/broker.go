package agreement

import (
	"bytes"
	"encoding/binary"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchAgreement is a helper to minimize the wiring of TopicListeners,
// collector and channels. The agreement component notarizes the new blocks after having collected a quorum of votes
func LaunchAgreement(eventBus *wire.EventBus, committee committee.Committee, keys *user.Keys,
	currentRound uint64) *broker {
	broker := newBroker(eventBus, committee, keys)
	broker.updateRound(currentRound)
	go broker.Listen()
	return broker
}

type broker struct {
	publisher   wire.EventPublisher
	handler     *agreementHandler
	state       consensus.State
	filter      *consensus.EventFilter
	accumulator *consensus.Accumulator
}

func launchFilter(eventBroker wire.EventBroker, committee committee.Committee,
	handler consensus.EventHandler, state consensus.State,
	accumulator *consensus.Accumulator) *consensus.EventFilter {
	filter := consensus.NewEventFilter(handler, state, accumulator, false)
	republisher := consensus.NewRepublisher(eventBroker, topics.Agreement)
	eventBroker.SubscribeCallback(string(topics.Agreement), filter.Collect)
	eventBroker.RegisterPreprocessor(string(topics.Agreement), republisher, &consensus.Validator{})
	return filter
}

func newBroker(eventBroker wire.EventBroker, committee committee.Committee, keys *user.Keys) *broker {
	handler := newHandler(committee, keys)
	accumulator := consensus.NewAccumulator(handler, consensus.NewAccumulatorStore())
	state := consensus.NewState()
	filter := launchFilter(eventBroker, committee, handler,
		state, accumulator)
	return &broker{
		publisher:   eventBroker,
		handler:     handler,
		state:       state,
		filter:      filter,
		accumulator: accumulator,
	}
}

// Listen for results coming from the accumulator and updating the round accordingly
func (b *broker) Listen() {
	for {
		evs := <-b.accumulator.CollectedVotesChan
		b.publishWinningHash(evs)
		b.updateRound(b.state.Round() + 1)
	}
}

func (b *broker) sendAgreement(m *bytes.Buffer) error {
	if !b.handler.AmMember(b.state.Round(), b.state.Step()) {
		return nil
	}

	unmarshaller := reduction.NewUnMarshaller()
	voteSet, err := unmarshaller.UnmarshalVoteSet(m)
	if err != nil {
		log.WithField("process", "agreement").WithError(err).Errorln("problem unmarshalling voteset")
		return err
	}

	msg, err := b.handler.createAgreement(voteSet, b.state.Round(), b.state.Step())
	if err != nil {
		log.WithField("process", "agreement").WithError(err).Errorln("problem creating agreement vote")
		return err
	}

	b.publisher.Stream(string(topics.Gossip), msg)
	return nil
}

func (b *broker) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "agreement",
		"round":   round,
	}).Debugln("updating round")
	b.filter.UpdateRound(round)
	b.publishRoundUpdate(round)
	b.accumulator.Clear()
	b.filter.FlushQueue()
}

func (b *broker) publishWinningHash(evs []wire.Event) {
	aev := evs[0].(*Agreement)
	b.publisher.Publish(msg.WinningBlockTopic, bytes.NewBuffer(aev.BlockHash))
}

// publishRoundUpdate publishes the new round in the EventPublisher
func (b *broker) publishRoundUpdate(round uint64) {
	// Marshalling the round update
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, round)
	buf := bytes.NewBuffer(bs)
	// publishing to the EventBus
	b.publisher.Publish(msg.RoundUpdateTopic, buf)
}
