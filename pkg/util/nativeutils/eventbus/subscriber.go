package eventbus

import lg "github.com/sirupsen/logrus"

// Subscriber subscribes a channel to Event notifications on a specific topic
type Subscriber interface {
	Subscribe(string, Listener) uint32
	Unsubscribe(string, uint32)
}

// Subscribe subscribes to a topic with a channel.
func (bus *EventBus) Subscribe(topic string, listener Listener) uint32 {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	return bus.listeners.Store(topic, listener)
}

// Unsubscribe removes all listeners defined for a topic.
func (bus *EventBus) Unsubscribe(topic string, id uint32) {
	found := bus.listeners.Delete(topic, id)

	logEB.WithFields(lg.Fields{
		"found": found,
		"topic": topic,
	}).Traceln("unsubscribing")
}
