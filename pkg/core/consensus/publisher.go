package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Publisher is used to direct consensus messages from the peer.MessageProcessor
// to the consensus components.
type Publisher struct {
	publisher eventbus.Publisher
}

// NewPublisher returns an initialized Publisher.
func NewPublisher(publisher eventbus.Publisher) *Publisher {
	return &Publisher{publisher}
}

// Process incoming consensus messages.
// Satisfies the peer.ProcessorFunc interface.
func (p *Publisher) Process(msg message.Message) ([]bytes.Buffer, error) {
	p.publisher.Publish(msg.Category(), msg)
	return nil, nil
}
