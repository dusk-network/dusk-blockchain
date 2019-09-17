package agreement

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func Launch(eventBroker wire.EventBroker, c committee.Foldable, keys user.Keys) {
	if c == nil {
		c = committee.NewAgreement()
	}
	broker := newBroker(eventBroker, c, keys)
	currentRound := getInitialRound(eventBroker)
	broker.updateRound(currentRound)
	go broker.Listen()
}

type broker struct {
	publisher  wire.EventPublisher
	handler    *agreementHandler
	state      consensus.State
	filter     *consensus.EventFilter
	resultChan <-chan voteSet
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
		publisher:  eventBroker,
		handler:    handler,
		state:      state,
		filter:     filter,
		resultChan: initReductionResultCollector(eventBroker),
	}

	return b
}

// Listen for results coming from the accumulator and updating the round accordingly
func (b *broker) Listen() {
	for {
		select {
		case evs := <-b.filter.Accumulator.CollectedVotesChan:
			b.publishEvent(evs)
			b.publishWinningHash(evs)
			b.updateRound(b.state.Round() + 1)
		case voteSet := <-b.resultChan:
			b.sendAgreement(voteSet)
		}
	}
}

func (b *broker) sendAgreement(voteSet voteSet) error {
	if voteSet.round != b.state.Round() {
		return errors.New("received results message is from a different round")
	}

	// We always increment the step, even if we are not included in the committee.
	// This way, we are always on the same step as everybody else.
	defer b.state.IncrementStep()

	// Make sure we actually received a voteset, before trying to aggregate it.
	if voteSet.votes == nil {
		// If we didn't, return here, so that we increment the step to stay in sync.
		log.WithField("process", "agreement").Debugln("received an empty voteset")
		return nil
	}

	if b.handler.AmMember(b.state.Round(), b.state.Step()) {
		msg, err := b.handler.createAgreement(voteSet.votes, b.state.Round(), b.state.Step())
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
