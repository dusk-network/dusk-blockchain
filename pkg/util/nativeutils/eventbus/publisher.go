package eventbus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
)

// Publisher publishes serialized messages on a specific topic
type Publisher interface {
	Publish(topics.Topic, message.Message) []error
}

// Publish executes callback defined for a topic.
// topic is explicitly set as it might be different from the message Category
// (i.e. in the Gossip case)
// Publishing is a fire and forget. If there is no listener for a topic, the
// messages are lost
func (bus *EventBus) Publish(topic topics.Topic, m message.Message) (errorList []error) {
	//logEB.WithFields(logrus.Fields{
	//	"topic":    topic,
	//	"category": m.Category(),
	//}).Traceln("publishing on the eventbus")

	// first serve the default topic listeners as they are most likely to need more time to process topics
	go func() {
		newErrList := bus.defaultListener.Forward(topic, m)
		diagnostics.LogPublishErrors("eventbus/publisher.go, Publish", newErrList)
	}()

	listeners := bus.listeners.Load(topic)
	for _, listener := range listeners {
		if err := listener.Notify(m); err != nil {
			logEB.
				WithError(err).
				WithField("topic", topic.String()).
				WithField("id", m.Id()).
				Warnln("listener failed to notify buffer")
			errorList = append(errorList, err)
		}
	}
	return errorList
}
