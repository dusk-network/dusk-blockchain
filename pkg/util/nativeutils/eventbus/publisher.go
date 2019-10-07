package eventbus

import (
	"bytes"
	"fmt"
)

// Publisher publishes serialized messages on a specific topic
type Publisher interface {
	Publish(string, *bytes.Buffer)
	//Stream(string, *bytes.Buffer)
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer *bytes.Buffer) {
	if messageBuffer == nil {
		err := fmt.Errorf("got a nil message on topic %s", topic)
		logEB.WithError(err).Errorln("preprocessor error")
		return
	}

	if err := bus.Preprocess(topic, messageBuffer); err != nil {
		logEB.WithError(err).Errorln("preprocessor error")
		return
	}

	bus.publish(topic, *messageBuffer)
}

func (bus *EventBus) publish(topic string, event bytes.Buffer) {

	// first serve the default topic listeners as they are most likely to need more time to (pre-)process topics
	go bus.defaultListener.Notify(topic, event)

	if listeners := bus.listeners.Load(topic); listeners != nil {
		for _, listener := range listeners {
			if err := listener.Notify(event); err != nil {
				logEB.WithField("topic", topic).Warnln("listener failed to notidy buffer")
			}
		}
	}
}

/*
// Stream a buffer to the subscribers for a specific topic.
func (bus *EventBus) Stream(topic string, messageBuffer *bytes.Buffer) {
	// TODO: this is probably a byproduct of the TopicListener
	processedMsg, err := bus.preprocess(topic, messageBuffer)
	if err != nil {
		logEB.WithError(err).WithField("topic", topic).Errorln("preprocessor error")
		return
	}

	// The listeners are simply a means to avoid memory leaks.
	listeners := bus.StreamListeners.Load(topic)
	for _, listener := range listeners {
		if err := listener.Publish(*processedMsg); err != nil {
			logEB.WithError(err).WithField("topic", topic).Debugln("cannot publish event on streamlistener")
			continue
		}
	}
}
*/
