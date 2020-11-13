package candidate

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	lg "github.com/sirupsen/logrus"
)

var log = lg.WithField("process", "candidate-publisher")

// Publisher is a component which simply notifies the receipt of Candidate
// messages on the event bus. Since Candidate messages will only arrive on
// topics.Candidate when specifically asked for, using `topics.GetCandidate`,
// this component basically serves to deliver responses to that message.
type Publisher struct {
	publisher eventbus.Publisher
}

// NewBroker returns an initialized Publisher struct.
func NewBroker(publisher eventbus.Publisher) *Publisher {
	return &Publisher{
		publisher: publisher,
	}
}

// Process a received Candidate message. Invalid Candidate messages are discarded.
func (b *Publisher) Process(msg message.Message) ([]*bytes.Buffer, error) {
	// We only publish valid candidate messages
	if err := Validate(msg); err != nil {
		return nil, err
	}

	b.publisher.Publish(topics.Candidate, msg)
	return nil, nil
}
