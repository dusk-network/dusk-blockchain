package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*agreement)(nil)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func newComponent(publisher eventbus.Publisher, keys user.Keys) *agreement {
	return newAgreement(publisher, keys)
}

type agreement struct {
	publisher   eventbus.Publisher
	handler     *handler
	accumulator *Accumulator
	keys        user.Keys
}

func newAgreement(publisher eventbus.Publisher, keys user.Keys) *agreement {
	return &agreement{
		publisher: publisher,
		keys:      keys,
	}
}

func (a *agreement) Initialize(store consensus.Store, r consensus.RoundUpdate) []consensus.Subscriber {
	a.handler = newHandler(a.keys, r.P)
	a.accumulator = newAccumulator(a.handler)
	agreementSubscriber := consensus.Subscriber{
		consensus.NewFilteringListener(a.CollectAgreementEvent, a.Filter),
		topics.Agreement,
	}

	go a.listen()
	return []consensus.Subscriber{agreementSubscriber}
}

func (a *agreement) Filter(hdr header.Header) bool {
	return !a.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (a *agreement) CollectAgreementEvent(event consensus.Event) error {

	ev := New(event.Header)
	if err := Unmarshal(&event.Payload, ev); err != nil {
		return err
	}

	a.accumulator.Process(*ev)
	return nil
}

// SetStep implements Component
func (a *agreement) SetStep(step uint8) {}

// Listen for results coming from the accumulator
func (a *agreement) listen() {
	evs := <-a.accumulator.CollectedVotesChan
	a.publishAgreement(evs[0])
}

func (a *agreement) publishAgreement(aev Agreement) {
	buf := new(bytes.Buffer)
	if err := Marshal(buf, aev); err != nil {
		log.WithFields(log.Fields{
			"process": "agreement",
			"error":   err,
		}).Errorln("could not marshal agreement event")
		return
	}

	a.publisher.Publish(topics.AgreementEvent, buf)
	a.publisher.Publish(topics.WinningBlockHash, bytes.NewBuffer(aev.BlockHash))
}

func (a *agreement) Finalize() {
	a.accumulator.Stop()
}
