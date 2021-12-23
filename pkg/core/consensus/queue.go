// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"sort"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/sirupsen/logrus"
)

const (
	maxMessages = 4096
)

// Queue is a Queue of Events grouped by rounds and steps. It is thread-safe
// through a sync.RWMutex.
// TODO: entries should become buntdb instead.
type Queue struct {
	lock    sync.RWMutex
	entries map[uint64]map[uint8][]message.Message
	items   int
}

// NewQueue creates a new Queue. It is primarily used by Collectors to
// temporarily store messages not yet relevant to the collection process.
func NewQueue() *Queue {
	entries := make(map[uint64]map[uint8][]message.Message)
	return &Queue{
		entries: entries,
	}
}

// GetEvents returns the events for a round and step.
func (eq *Queue) GetEvents(round uint64, step uint8) []message.Message {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	if eq.entries[round][step] != nil {
		messages := eq.entries[round][step]

		eq.entries[round][step] = nil
		eq.items -= len(messages)
		return messages
	}

	return nil
}

// PutEvent stores an Event at a given round and step.
func (eq *Queue) PutEvent(round uint64, step uint8, m message.Message) {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	if eq.items >= maxMessages {
		logrus.WithField("process", "consensus").
			WithField("round", round).
			WithField("step", step).
			WithField("topic", m.Category()).
			Warnln("dropping message, queue has reached max capacity")
		return
	}

	// Initialize the map on this round if it was not yet created
	if eq.entries[round] == nil {
		eq.entries[round] = make(map[uint8][]message.Message)
	}

	// Initialize the array on this step if it was not yet created
	if eq.entries[round][step] == nil {
		eq.entries[round][step] = make([]message.Message, 0)
	}

	eq.entries[round][step] = append(eq.entries[round][step], m)
	eq.items++
}

// Clear the queue. This method swaps the internal `entries` map, to avoid
// a situation where memory is continuously allocated and never freed.
func (eq *Queue) Clear(round uint64) {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	newEntries := make(map[uint64]map[uint8][]message.Message)
	eq.items = 0

	for r := range eq.entries {
		if r > round {
			newEntries[r] = make(map[uint8][]message.Message)

			for s, m := range eq.entries[r] {
				newEntries[r][s] = m
				eq.items += len(m)

				delete(eq.entries[r], s)
			}

			delete(eq.entries, r)
		}
	}

	eq.entries = newEntries
}

// Flush all events stored for a specific round from the queue, and return them
// Messages are sorted giving priority to AggrAgreement and then to their step.
func (eq *Queue) Flush(round uint64) []message.Message {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	if eq.entries[round] != nil {
		events := make([]message.Message, 0)
		evround := eq.entries[round]

		steps := make([]uint8, 0, len(evround))
		for k := range evround {
			steps = append(steps, k)
		}
		// Give priority to oldest steps
		sort.SliceStable(steps, func(i, j int) bool {
			return steps[i] < steps[j]
		})

		for _, step := range steps {
			events = append(events, evround[step]...)
			eq.entries[round][step] = nil
		}
		// Give priority to AggrAgreement messages otherwise use previous step order
		sort.SliceStable(events, func(i, j int) bool {
			topic_i := events[i].Category()
			if topic_i == topics.AggrAgreement {
				if events[j].Category() != topics.AggrAgreement {
					return true
				}
				return i < j
			}
			if events[j].Category() != topics.AggrAgreement {
				return false
			}
			return i < j
		})

		eq.items -= len(events)
		return events
	}

	return nil
}
