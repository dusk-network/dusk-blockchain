package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Launch is a helper to minimize the wiring of TopicListeners, collector and
// channels. The agreement component notarizes the new blocks after having
// collected a quorum of votes
func NewComponent(publisher eventbus.Publisher, keys user.Keys, p user.Provisioners) *broker {
	b := newBroker(publisher, keys, p, requestStepUpdate)
	go b.listen()
	return b
}

// TODO: change naming to agreement for method receivers
type broker struct {
	broker      eventbus.Publisher
	handler     *agreementHandler
	accumulator *consensus.Accumulator
}

func newBroker(publisher eventbus.Publisher, keys user.Keys, p user.Provisioners) *broker {
	handler := newHandler(keys, p)
	b := &broker{
		publisher: publisher,
		handler:   handler,
	}
	return b
}

func (b *broker) Initialize() ([]consensus.Subscriber, consensus.EventHandler) {
	agreementSubscriber := consensus.Subscriber{
		topic:    topics.Agreement,
		listener: consensus.NewCallbackListener(b.CollectAgreementEvent),
	}

	return []Subscriber{agreementSubscriber}, b.handler
}

// Listen for results coming from the accumulator
func (b *broker) listen() {
	for {
		select {
		case evs := <-b.filter.Accumulator.CollectedVotesChan:
			b.publishEvent(evs)
			b.publishWinningHash(evs)
		}
	}
}

func (b *broker) publishWinningHash(evs []wire.Event) {
	aev := evs[0].(*Agreement)
	b.broker.Publish(topics.WinningBlockHash, bytes.NewBuffer(aev.BlockHash))
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

	b.broker.Publish(topics.AgreementEvent, buf)
}

func (b *broker) Finalize() {
	b.broker.Unsubscribe(topics.Agreement, b.agreementID)
}
