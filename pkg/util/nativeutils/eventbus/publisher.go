package eventbus

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Publisher publishes serialized messages on a specific topic
type Publisher interface {
	Publish(topics.Topic, *bytes.Buffer)
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic topics.Topic, messageBuffer *bytes.Buffer) {
	if messageBuffer == nil {
		err := fmt.Errorf("got a nil message on topic %s", topic)
		logEB.WithField("topic", topic.String()).WithError(err).Errorln("preprocessor error")
		return
	}

	bus.publish(topic, *messageBuffer)
}

func (bus *EventBus) publish(topic topics.Topic, event bytes.Buffer) {

	// first serve the default topic listeners as they are most likely to need more time to (pre-)process topics
	go bus.defaultListener.Notify(topic, event)

	if listeners := bus.listeners.Load(topic); listeners != nil {
		for _, listener := range listeners {
			if err := listener.Notify(event); err != nil {
				logEB.WithError(err).WithField("topic", topic).Warnln("listener failed to notify buffer")
			}
		}
	}
}
