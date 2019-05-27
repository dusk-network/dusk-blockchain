package consensus

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// AccumulatorStore is a helper struct for common operations on stored Event Arrays
// AccumulatorStore is an helper for common operations on stored Event Arrays
type (
	AccumulatorStore struct {
		lock  sync.RWMutex
		evMap map[string][]wire.Event
	}
)

// NewAccumulatorStore returns an initialized AccumulatorStore.
func NewAccumulatorStore() *AccumulatorStore {
	return &AccumulatorStore{
		evMap: make(map[string][]wire.Event),
	}
}

// Clear up the AccumulatorStore.
func (sec *AccumulatorStore) Clear() {
	sec.lock.Lock()
	defer sec.lock.Unlock()
	sec.evMap = make(map[string][]wire.Event)
}

// Contains checks if we already collected this event
func (sec *AccumulatorStore) Contains(event wire.Event, identifier string) bool {
	sec.lock.RLock()
	defer sec.lock.RUnlock()
	for _, stored := range sec.evMap[identifier] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Insert the Event keeping track of the identifier (step, block hash, voted hash)
// it belongs to. It silently ignores duplicates (meaning it does not store an event
// in case it is already found at the identifier specified). It returns the number of
// events stored at specified identifier *after* the store operation
func (sec *AccumulatorStore) Insert(event wire.Event, identifier string) int {
	sec.lock.RLock()
	eventList := sec.evMap[identifier]
	sec.lock.RUnlock()
	if sec.Contains(event, identifier) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 100)
	}

	// storing the agreement vote for the proper identifier
	eventList = append(eventList, event)
	sec.lock.Lock()
	sec.evMap[identifier] = eventList
	sec.lock.Unlock()
	return len(eventList)
}

// Get a set of Events, stored under a specified identifier.
func (sec *AccumulatorStore) Get(identifier string) []wire.Event {
	sec.lock.RLock()
	defer sec.lock.RUnlock()
	return sec.evMap[identifier]
}

// All returns all of the Events currently in the AccumulatorStore.
func (sec *AccumulatorStore) All() []wire.Event {
	allEvents := make([]wire.Event, 0)
	sec.lock.RLock()
	defer sec.lock.RUnlock()
	for _, evs := range sec.evMap {
		allEvents = append(allEvents, evs...)
	}

	return allEvents
}
