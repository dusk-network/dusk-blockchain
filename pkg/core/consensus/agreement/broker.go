package agreement

import (
	"bytes"
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func Launch(eventBroker wire.EventBroker, c committee.Foldable, keys user.Keys) {
	if c == nil {
		c = committee.NewAgreement(eventBroker, nil)
	}
	broker := newBroker(eventBroker, c, keys)
	currentRound := getInitialRound(eventBroker)
	broker.updateRound(currentRound)
}

type broker struct {
	publisher wire.EventPublisher
	handler   *agreementHandler
	state     consensus.State
	filter    *consensus.EventFilter
}

func launchFilter(eventBroker wire.EventBroker, committee committee.Committee,
	handler consensus.AccumulatorHandler, state consensus.State) *consensus.EventFilter {
	filter := consensus.NewEventFilter(handler, state, false)
	republisher := consensus.NewRepublisher(eventBroker, topics.Agreement)
	eventBroker.SubscribeCallback(string(topics.Agreement), filter.Collect)
	eventBroker.RegisterPreprocessor(string(topics.Agreement), republisher, &consensus.Validator{})
	return filter
}

func newBroker(eventBroker wire.EventBroker, committee committee.Foldable, keys user.Keys) *broker {
	handler := newHandler(committee, keys)
	state := consensus.NewState()
	filter := launchFilter(eventBroker, committee, handler, state)
	b := &broker{
		publisher: eventBroker,
		handler:   handler,
		state:     state,
		filter:    filter,
	}

	eventBroker.SubscribeCallback(msg.ReductionResultTopic, b.sendAgreement)
	return b
}

// Listen for results coming from the accumulator and updating the round accordingly
func (b *broker) Listen() {
	evs := <-b.filter.Accumulator.CollectedVotesChan
	b.publishEvent(evs)
	b.publishWinningHash(evs)
	b.updateRound(b.state.Round() + 1)
}

func (b *broker) sendAgreement(m *bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64(m, binary.LittleEndian, &round); err != nil {
		return err
	}

	if round != b.state.Round() {
		return errors.New("received results message is from a different round")
	}

	// We always increment the step, even if we are not included in the committee.
	// This way, we are always on the same step as everybody else.
	defer b.state.IncrementStep()

	if b.handler.AmMember(b.state.Round(), b.state.Step()) {
		unmarshaller := reduction.NewUnMarshaller()
		voteSet, err := unmarshaller.UnmarshalVoteSet(m)
		if err != nil {
			log.WithField("process", "agreement").WithError(err).Warnln("problem unmarshalling voteset")
			return err
		}

		msg, err := b.handler.createAgreement(voteSet, b.state.Round(), b.state.Step())
		if err != nil {
			log.WithField("process", "agreement").WithError(err).Errorln("problem creating agreement vote")
			return err
		}

		b.publisher.Stream(string(topics.Gossip), msg)
	}

	return nil
}

func (b *broker) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "agreement",
		"round":   round,
	}).Debugln("updating round")
	b.filter.UpdateRound(round)
	consensus.UpdateRound(b.publisher, round)
	b.filter.FlushQueue()
	go b.Listen()
}

func (b *broker) publishWinningHash(evs []wire.Event) {
	aev := evs[0].(*Agreement)
	b.publisher.Publish(msg.WinningBlockHashTopic, bytes.NewBuffer(aev.BlockHash))
}

func (b *broker) publishEvent(evs []wire.Event) {
	marshaller := NewUnMarshaller()
	buf := new(bytes.Buffer)
	if err := marshaller.Marshal(buf, evs[0]); err != nil {
		log.WithFields(log.Fields{
			"process": "agreement",
			"error":   err,
		}).Errorln("could not marshal agreement event")
		return
	}

	b.publisher.Publish(msg.AgreementEventTopic, buf)
}
