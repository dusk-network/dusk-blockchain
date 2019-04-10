package consensus

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

// EventQueue is a Queue of Events grouped by rounds and steps. It is threadsafe through a sync.RWMutex
type EventQueue struct {
	entries map[uint64]map[uint8][]wire.Event
}

// NewEventQueue creates a new EventQueue. It is primarily used by Collectors to temporarily store messages not yet relevant to the collection process
func NewEventQueue() *EventQueue {
	entries := make(map[uint64]map[uint8][]wire.Event)
	return &EventQueue{
		entries,
	}
}

// GetEvents returns the events for a round and step
func (s *EventQueue) GetEvents(round uint64, step uint8) []wire.Event {
	if s.entries[round][step] != nil {
		messages := s.entries[round][step]
		s.entries[round][step] = nil
		return messages
	}

	return nil
}

// PutEvent stores an Event at a given round and step
func (s *EventQueue) PutEvent(round uint64, step uint8, m wire.Event) {
	// Initialise the map on this round if it was not yet created
	if s.entries[round] == nil {
		s.entries[round] = make(map[uint8][]wire.Event)
	}

	s.entries[round][step] = append(s.entries[round][step], m)
}

// Clear the queue
func (s *EventQueue) Clear(round uint64) {
	s.entries[round] = nil
}

// ConsumeNextStepEvents retrieves the Events stored at the lowest step for a passed round and returns them. The step gets deleted
func (s *EventQueue) ConsumeNextStepEvents(round uint64) ([]wire.Event, uint8) {
	steps := s.entries[round]
	if steps == nil {
		return nil, 0
	}

	nextStep := uint8(0)
	for k := range steps {
		if k > nextStep {
			nextStep = k
		}
	}

	events := s.entries[round][nextStep]
	delete(s.entries[round], nextStep)
	return events, nextStep
}

// ConsumeUntil consumes Events until the round specified (excluded). It returns the map slice deleted
func (s *EventQueue) ConsumeUntil(round uint64) map[uint64]map[uint8][]wire.Event {
	ret := make(map[uint64]map[uint8][]wire.Event)
	for k := range s.entries {
		if k < round {
			ret[k] = s.entries[k]
		}
		delete(s.entries, k)
	}
	return ret
}
