package agreement

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func Launch(eventBroker eventbus.Broker, keys user.Keys) {
	b := newBroker(eventBroker, keys)
	go func(b *broker) {
		roundUpdate := <-b.roundChan
		b.updateRound(roundUpdate)
		b.Listen()
	}(b)
}

type broker struct {
	publisher  eventbus.Publisher
	handler    *agreementHandler
	state      consensus.State
	filter     *consensus.EventFilter
	resultChan <-chan voteSet
	roundChan  <-chan consensus.RoundUpdate
}

func launchFilter(eventBroker eventbus.Broker, handler consensus.AccumulatorHandler, state consensus.State) *consensus.EventFilter {
	filter := consensus.NewEventFilter(handler, state, false)
	republisher := consensus.NewRepublisher(eventBroker, topics.Agreement)
	eventBroker.SubscribeCallback(string(topics.Agreement), filter.Collect)
	eventBroker.RegisterPreprocessor(string(topics.Agreement), republisher, &consensus.Validator{})
	return filter
}

func newBroker(eventBroker eventbus.Broker, keys user.Keys) *broker {
	handler := newHandler(keys)
	state := consensus.NewState()
	filter := launchFilter(eventBroker, handler, state)
	b := &broker{
		publisher:  eventBroker,
		handler:    handler,
		state:      state,
		filter:     filter,
		resultChan: initReductionResultCollector(eventBroker),
		roundChan:  consensus.InitRoundUpdate(eventBroker),
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
		case voteSet := <-b.resultChan:
			if err := b.sendAgreement(voteSet); err != nil {
				log.WithField("process", "agreement").WithError(err).Warnln("sending agreement failed")
			}
		case roundUpdate := <-b.roundChan:
			b.updateRound(roundUpdate)
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

func (b *broker) updateRound(roundUpdate consensus.RoundUpdate) {
	log.WithFields(log.Fields{
		"process": "agreement",
		"round":   roundUpdate.Round,
	}).Debugln("updating round")
	b.filter.UpdateRound(roundUpdate.Round)
	b.handler.UpdateProvisioners(roundUpdate.P)
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
