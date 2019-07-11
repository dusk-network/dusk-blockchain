package consensus

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// EventQueue is a Queue of Events grouped by rounds and steps. It is threadsafe
// through a sync.RWMutex.
type EventQueue struct {
	lock    sync.RWMutex
	entries map[uint64]map[uint8][]wire.Event
}

// NewEventQueue creates a new EventQueue. It is primarily used by Collectors to
// temporarily store messages not yet relevant to the collection process.
func NewEventQueue() *EventQueue {
	entries := make(map[uint64]map[uint8][]wire.Event)
	return &EventQueue{
		entries: entries,
	}
}

// GetEvents returns the events for a round and step.
func (eq *EventQueue) GetEvents(round uint64, step uint8) []wire.Event {
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
func (eq *EventQueue) PutEvent(round uint64, step uint8, m wire.Event) {
	eq.lock.Lock()
	defer eq.lock.Unlock()

	// Initialise the map on this round if it was not yet created
	if eq.entries[round] == nil {
		eq.entries[round] = make(map[uint8][]wire.Event)
	}

	// Initialise the array on this step if it was not yet created
	if eq.entries[round][step] == nil {
		eq.entries[round][step] = make([]wire.Event, 0)
	}

	eq.entries[round][step] = append(eq.entries[round][step], m)
}

// Clear the queue.
func (eq *EventQueue) Clear(round uint64) {
	eq.lock.Lock()
	defer eq.lock.Unlock()
	eq.entries[round] = nil
}

// Flush all events stored for a specific round from the queue, and return them.
func (eq *EventQueue) Flush(round uint64) []wire.Event {
	eq.lock.Lock()
	defer eq.lock.Unlock()
	events := make([]wire.Event, 0)
	if eq.entries[round] != nil {
		for step, evs := range eq.entries[round] {
			events = append(events, evs...)
			eq.entries[round][step] = nil
		}
		return events
	}
	return nil
}
