// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
)

// Publisher publishes serialized messages on a specific topic.
type Publisher interface {
	Publish(topics.Topic, message.Message) []error
}

// Publish executes callback defined for a topic.
// topic is explicitly set as it might be different from the message Category
// (i.e. in the Gossip case).
// Publishing is a fire and forget. If there is no listener for a topic, the
// messages are lost.
// FIXME: Publish should fail fast and return one error. Since the code is largely
// asynchronous, we don't expect errors and if they happen, this should be
// reported asap.
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
			errorList = append(errorList, err)
		}
	}
	return errorList
}
