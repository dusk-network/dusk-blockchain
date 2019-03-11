package wire

import (
	"bytes"
	"fmt"
	"math/rand"
)

// EventBus - box for handlers and callbacks.
type EventBus struct {
	handlers    map[string]*eventHandler
	broadcaster []*eventHandler
}

type eventHandler struct {
	id             uint32
	messageChannel chan<- *bytes.Buffer
}

// New returns new EventBus with empty handlers.
func New() *EventBus {
	return &EventBus{
		make(map[string]*eventHandler),
		nil,
	}
}

// doRegister handles the registration logic and is utilized by the public Register functions
func (bus *EventBus) doRegister(topic string, handler *eventHandler) {
	bus.handlers[topic] = handler
}

// Register registers to a topic with an asynchronous callback.
func (bus *EventBus) Register(topic string, messageChannel chan<- *bytes.Buffer) {
	id := rand.Uint32()
	bus.doRegister(topic, &eventHandler{
		id, messageChannel,
	})
}

// RegisterAll registers to all topics, and returns a unique ID associated
// to the event handler the function creates. The ID can later be used to
// unregister from all topics.
func (bus *EventBus) RegisterAll(messageChannel chan<- *bytes.Buffer) uint32 {
	id := rand.Uint32()
	bus.broadcaster = append(bus.broadcaster, &eventHandler{
		id, messageChannel,
	})

	return id
}

// UnregisterAll will unregister from all topics. The function takes an ID
// which points to the handler associated to the caller.
func (bus *EventBus) UnregisterAll(id uint32) {
	for i, handler := range bus.broadcaster {
		if handler.id == id {
			bus.broadcaster = append(bus.broadcaster[:i], bus.broadcaster[i+1:]...)
		}
	}
}

// HasCallback returns true if exists any callback registered to the topic.
func (bus *EventBus) HasCallback(topic string) bool {
	_, ok := bus.handlers[topic]
	return ok
}

// Unregister removes callback defined for a topic.
// Returns error if there are no callbacks registered to the topic.
func (bus *EventBus) Unregister(topic string) error {
	if _, ok := bus.handlers[topic]; ok {
		delete(bus.handlers, topic)
		return nil
	}
	return fmt.Errorf("topic %s doesn't exist", topic)
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer *bytes.Buffer) {
	if handler, ok := bus.handlers[topic]; ok {
		go bus.doPublish(handler, messageBuffer)
	}

	for _, handler := range bus.broadcaster {
		go bus.doPublish(handler, messageBuffer)
	}
}

func (bus *EventBus) doPublish(handler *eventHandler, messageBuffer *bytes.Buffer) {
	handler.messageChannel <- messageBuffer
}
