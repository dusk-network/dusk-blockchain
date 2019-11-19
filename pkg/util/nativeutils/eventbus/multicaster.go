package eventbus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Multicaster allows for a single Listener to listen to multiple topics
type Multicaster interface {

	AddDefaultTopic(topics.Topic)
	SubscribeDefault(Listener) uint32
}

// AddDefaultTopic adds a topic to the default multiListener
func (bus *EventBus) AddDefaultTopic(topic topics.Topic) {
	bus.defaultListener.Add([]byte{byte(topic)})
}

// SubscribeDefault subscribes a Listener to the default multiListener.
// This is normally useful for implementing a sub-dispatching mechanism
// (i.e. bus of busses architecture)
func (bus *EventBus) SubscribeDefault(listener Listener) uint32 {
	return bus.defaultListener.Store(listener)
}
