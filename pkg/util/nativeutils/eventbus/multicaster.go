// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Multicaster allows for a single Listener to listen to multiple topics
type Multicaster interface {
	AddDefaultTopic(topics.Topic)
	SubscribeDefault(Listener) uint32
}

// AddDefaultTopic add topics to the default multiListener
func (bus *EventBus) AddDefaultTopic(tpcs ...topics.Topic) {
	for _, tpc := range tpcs {
		bus.defaultListener.Add(tpc)
	}
}

// SubscribeDefault subscribes a Listener to the default multiListener.
// This is normally useful for implementing a sub-dispatching mechanism
// (i.e. bus of busses architecture)
func (bus *EventBus) SubscribeDefault(listener Listener) uint32 {
	return bus.defaultListener.Store(listener)
}
