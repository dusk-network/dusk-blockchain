package wire

import (
	"bytes"
	"math/rand"
	"sync"

	log "github.com/sirupsen/logrus"
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

// NewEventBus returns new EventBus with empty handlers.
func NewEventBus() *EventBus {
	return &EventBus{
		sync.RWMutex{},
		make(map[string][]*eventHandler),
		nil,
	}
}

// subscribe handles the subscription logic and is utilized by the public
// Subscribe functions
func (bus *EventBus) subscribe(topic string, handler *eventHandler) {
	bus.handlers[topic] = append(bus.handlers[topic], handler)
}

// Subscribe subscribes to a topic with an asynchronous callback.
func (bus *EventBus) Subscribe(topic string, messageChannel chan<- *bytes.Buffer) uint32 {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	id := rand.Uint32()
	bus.subscribe(topic, &eventHandler{
		id, messageChannel,
	})

	return id
}

// Unsubscribe removes a handler defined for a topic.
// Returns true if a handler is found with the id and the topic specified
func (bus *EventBus) Unsubscribe(topic string, id uint32) bool {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	return bus.unsubscribe(topic, id)
}

func (bus *EventBus) unsubscribe(topic string, id uint32) bool {
	if _, ok := bus.handlers[topic]; ok {
		for i, handler := range bus.handlers[topic] {
			if handler.id == id {
				bus.handlers[topic] = append(bus.handlers[topic][:i],
					bus.handlers[topic][i+1:]...)
				return true
			}
		}
	}
	return false
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

// UnsubscribeAll will unsubscribe from all topics and broadcasting channel. The function takes an ID
// which points to the handler associated to the caller. Wherever possible, `Unsubscribe` should be used instead as it is less intensive
func (bus *EventBus) UnsubscribeAll(id uint32) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	for topic := range bus.handlers {
		if ok := bus.unsubscribe(topic, id); ok {
			break
		}
	}
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
	return bus.hasHandler(topic)
}

func (bus *EventBus) hasHandler(topic string) bool {
	_, ok := bus.handlers[topic]
	return ok
}

// Publish executes callback defined for a topic.
func (bus *EventBus) Publish(topic string, messageBuffer *bytes.Buffer) {
	bus.busLock.Lock()
	defer bus.busLock.Unlock()
	if handlers, ok := bus.handlers[topic]; ok {
		bus.publish(handlers, messageBuffer, topic)
	}

	bus.publish(bus.broadcaster, messageBuffer, topic)
}

func (bus *EventBus) publish(handlers []*eventHandler, messageBuffer *bytes.Buffer, topic string) {
	for _, handler := range handlers {
		select {
		case handler.messageChannel <- messageBuffer:
		default:
			log.WithFields(log.Fields{
				"id":       handler.id,
				"topic":    topic,
				"process":  "eventbus",
				"capacity": len(handler.messageChannel),
			}).Debugln("handler.messageChannel buffer failed")
		}
	}
}
