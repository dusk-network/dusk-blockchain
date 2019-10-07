package eventbus

import "bytes"

// Multicaster allows for a single Listener to listen to multiple topics
type Multicaster interface {
	AddDefaultTopic(topic string)
	SubscribeDefault(callback func(m *bytes.Buffer) error) uint32
}

// AddDefaultTopic adds a topic to the default multiListener
func (bus *EventBus) AddDefaultTopic(topic string) {
	bus.defaultListener.Add([]byte(topic))
}

// SubscribeDefault subscribes a callback to the default multiListener.
// This is normally useful for implementing a sub-dispatching mechanism
// (i.e. bus of busses architecture)
func (bus *EventBus) SubscribeDefault(callback func(m bytes.Buffer) error) uint32 {
	return bus.defaultListener.Store(&CallbackListener{callback})
}
