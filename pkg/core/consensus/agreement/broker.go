package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func newComponent(publisher eventbus.Publisher, keys user.Keys) *agreement {
	return newAgreement(publisher, keys)
}

type agreement struct {
	publisher   eventbus.Publisher
	handler     *agreementHandler
	accumulator *consensus.Accumulator
	keys        user.Keys
}

func newAgreement(publisher eventbus.Publisher, keys user.Keys) *agreement {
	return &agreement{
		publisher: publisher,
		keys:      keys,
	}
}

func (a *agreement) Initialize(provisioners user.Provisioners) []consensus.Subscriber {
	a.handler = newHandler(a.keys, provisioners)
	a.accumulator = newAccumulator(a.handler, newAccumulatorStore())
	agreementSubscriber := &consensus.Subscriber{
		consensus.NewFilteringListener(a.CollectAgreementEvent, a.Filter),
		topics.Agreement,
	}

	go a.listen()
	return []Subscriber{agreementSubscriber}
}

func (a *agreement) Filter(hdr header.Header) bool {
	return !a.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

// SetStep implements Component
func (a *agreement) SetStep(step uint8) {}

// Listen for results coming from the accumulator
func (a *agreement) listen() {
	evs := <-a.filter.Accumulator.CollectedVotesChan
	a.publishEvent(evs)
	a.publishWinningHash(evs)
}

func (a *agreement) publishWinningHash(evs []wire.Event) {
	aev := evs[0].(*Agreement)
	a.publisher.Publish(topics.WinningBlockHash, bytes.NewBuffer(aev.BlockHash))
}

func (a *agreement) publishEvent(evs []wire.Event) {
	marshaller := NewUnMarshaller()
	buf := new(bytes.Buffer)
	if err := marshaller.Marshal(buf, evs[0]); err != nil {
		log.WithFields(log.Fields{
			"process": "agreement",
			"error":   err,
		}).Errorln("could not marshal agreement event")
		return
	}

	a.publisher.Publish(topics.AgreementEvent, buf)
}

func (a *agreement) Finalize() {
	a.accumulator.Stop()
}
