package wire

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
)

// EventBus - box for handlers and callbacks.
type EventBus struct {
	busLock     sync.RWMutex
	handlers    map[string][]*eventHandler
	broadcaster []*eventHandler
}

type eventHandler struct {
	id             uint32
	messageChannel chan<- *bytes.Buffer
}

// New returns new EventBus with empty handlers.
func New() *EventBus {
	return &EventBus{
		sync.RWMutex{},
		make(map[string][]*eventHandler),
		nil,
	}
}

// doSubscribe handles the subscription logic and is utilized by the public
// Subscribe functions
func (bus *EventBus) doSubscribe(topic string, handler *eventHandler) {
	bus.handlers[topic] = append(bus.handlers[topic], handler)
}

// Subscribe subscribes to a topic with an asynchronous callback.
func (bus *EventBus) Subscribe(topic string, messageChannel chan<- *bytes.Buffer) uint32 {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	id := rand.Uint32()
	bus.doSubscribe(topic, &eventHandler{
		id, messageChannel,
	})

	return id
}

// Unsubscribe removes a handler defined for a topic.
// Returns error if there are no handlers subscribed to the topic.
func (bus *EventBus) Unsubscribe(topic string, id uint32) error {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	if _, ok := bus.handlers[topic]; ok {
		for i, handler := range bus.handlers[topic] {
			if handler.id == id {
				bus.handlers[topic] = append(bus.handlers[topic][:i],
					bus.handlers[topic][i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("topic %s doesn't exist", topic)
}

// SubscribeAll subscribes to all topics, and returns a unique ID associated
// to the event handler the function creates. The ID can later be used to
// unsubscribe from all topics.
func (bus *EventBus) SubscribeAll(messageChannel chan<- *bytes.Buffer) uint32 {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	id := rand.Uint32()
	bus.broadcaster = append(bus.broadcaster, &eventHandler{
		id, messageChannel,
	})

	return id
}

// UnsubscribeAll will unsubscribe from all topics. The function takes an ID
// which points to the handler associated to the caller.
func (bus *EventBus) UnsubscribeAll(id uint32) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	for i, handler := range bus.broadcaster {
		if handler.id == id {
			bus.broadcaster = append(bus.broadcaster[:i], bus.broadcaster[i+1:]...)
		}
	}
}

// HasHandler returns true if exists any handler is subscribed to the topic.
func (bus *EventBus) HasHandler(topic string) bool {
	bus.busLock.RLock()
	defer bus.busLock.RUnlock()
	_, ok := bus.handlers[topic]
	return ok
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer *bytes.Buffer) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	if handlers, ok := bus.handlers[topic]; ok {
		go bus.doPublish(handlers, messageBuffer)
	}

	go bus.doPublish(bus.broadcaster, messageBuffer)
}

func (bus *EventBus) doPublish(handlers []*eventHandler, messageBuffer *bytes.Buffer) {
	for _, handler := range handlers {
		select {
		case handler.messageChannel <- messageBuffer:
		default:
			fmt.Printf("handler.messageChannel buffer failed for handler.id %d\n", handler.id)
		}
	}
}
