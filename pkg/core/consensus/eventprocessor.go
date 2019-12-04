package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	// Republisher is responsible for gossiping a received event buffer.
	Republisher struct {
		publisher eventbus.Publisher
		topic     topics.Topic
	}
)

// NewRepublisher returns a Republisher containing the specified parameters.
func NewRepublisher(publisher eventbus.Publisher, topic topics.Topic) *Republisher {
	return &Republisher{publisher, topic}
}

// Process propagates a received event buffer to other nodes in the network.
func (r *Republisher) Process(eventBuffer *bytes.Buffer) error {
	return r.process(*eventBuffer)
}

func (r *Republisher) process(eventBuffer bytes.Buffer) error {
	if err := topics.Prepend(&eventBuffer, r.topic); err != nil {
		return err
	}

	r.publisher.Publish(topics.Gossip, &eventBuffer)
	return nil
}
