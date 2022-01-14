// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	lg "github.com/sirupsen/logrus"
)

var _ Broker = (*EventBus)(nil)

var logEB = lg.WithField("process", "eventbus")

// TopicProcessor is the interface for preprocessing events belonging to a specific topic.
type (
	// Broker is an Publisher and an Subscriber.
	Broker interface {
		Subscriber
		Publisher
	}

	// EventBus - box for listeners and callbacks.
	EventBus struct {
		listeners       *listenerMap
		defaultListener *multiListener
	}
)

// New returns new EventBus with empty listeners.
func New() *EventBus {
	return &EventBus{
		listeners:       newListenerMap(),
		defaultListener: newMultiListener(),
	}
}

func (e *EventBus) Close() {
	e.listeners.Close()
	e.defaultListener.Close()
}
