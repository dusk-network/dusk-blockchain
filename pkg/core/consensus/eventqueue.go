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
func (s *EventQueue) GetEvents(round uint64, step uint8) []wire.Event {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.entries[round][step] != nil {
		messages := s.entries[round][step]
		s.entries[round][step] = nil
		return messages
	}

	return nil
}

// PutEvent stores an Event at a given round and step.
func (s *EventQueue) PutEvent(round uint64, step uint8, m wire.Event) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Initialise the map on this round if it was not yet created
	if s.entries[round] == nil {
		s.entries[round] = make(map[uint8][]wire.Event)
	}

	// Initialise the array on this step if it was not yet created
	if s.entries[round][step] == nil {
		s.entries[round][step] = make([]wire.Event, 0)
	}

	s.entries[round][step] = append(s.entries[round][step], m)
}

// Clear the queue.
func (s *EventQueue) Clear(round uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.entries[round] = nil
}

// Flush all events stored for a specific round from the queue, and return them.
func (s *EventQueue) Flush(round uint64) []wire.Event {
	s.lock.Lock()
	defer s.lock.Unlock()
	events := make([]wire.Event, 0)
	if s.entries[round] != nil {
		for step, evs := range s.entries[round] {
			events = append(events, evs...)
			s.entries[round][step] = nil
		}
		return events
	}
	return nil
}
