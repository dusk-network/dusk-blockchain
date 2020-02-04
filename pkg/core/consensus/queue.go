package consensus

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// Queue is a Queue of Events grouped by rounds and steps. It is threadsafe
// through a sync.RWMutex.
type Queue struct {
	lock    sync.RWMutex
	entries map[uint64]map[uint8][]message.Message
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
		return messages
	}

	return nil
}

// PutEvent stores an Event at a given round and step.
func (eq *Queue) PutEvent(round uint64, step uint8, m message.Message) {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	// Initialise the map on this round if it was not yet created
	if eq.entries[round] == nil {
		eq.entries[round] = make(map[uint8][]message.Message)
	}

	// Initialise the array on this step if it was not yet created
	if eq.entries[round][step] == nil {
		eq.entries[round][step] = make([]message.Message, 0)
	}

	eq.entries[round][step] = append(eq.entries[round][step], m)
}

// Clear the queue.
func (eq *Queue) Clear(round uint64) {
	eq.lock.Lock()
	defer eq.lock.Unlock()
	eq.entries[round] = nil
}

// Flush all events stored for a specific round from the queue, and return them.
func (eq *Queue) Flush(round uint64) []message.Message {
	eq.lock.Lock()
	defer eq.lock.Unlock()
	if eq.entries[round] != nil {
		events := make([]message.Message, 0)
		for step, evs := range eq.entries[round] {
			events = append(events, evs...)
			eq.entries[round][step] = nil
		}
		return events
	}
	return nil
}
