package eventbus

import (
	"sync"
	"time"

	lg "github.com/sirupsen/logrus"
)

var _ Broker = (*EventBus)(nil)

var napTime = 1 * time.Millisecond
var _signal struct{}
var logEB = lg.WithField("process", "eventbus")

// TopicProcessor is the interface for preprocessing events belonging to a specific topic
type (
	// Broker is an Publisher and an Subscriber
	Broker interface {
		Subscriber
		Publisher
	}

	// EventBus - box for listeners and callbacks.
	EventBus struct {
		busLock         sync.RWMutex
		listeners       *listenerMap
		defaultListener *multiListener
	}
)

// New returns new EventBus with empty listeners.
func New() *EventBus {
	return &EventBus{
		busLock:         sync.RWMutex{},
		listeners:       newListenerMap(),
		defaultListener: newMultiListener(),
	}
}
