package reduction

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

// reductionEventCollector is a helper for common operations on stored Event Arrays
type reductionEventCollector map[string][]wire.Event

// Clear up the Collector
func (rec reductionEventCollector) Clear() {
	for key := range rec {
		delete(rec, key)
	}
}

// Contains checks if we already collected this event
func (rec reductionEventCollector) Contains(e wire.Event, hash string) bool {
	for _, stored := range rec[hash] {
		if e.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the hash it belongs to.
func (rec reductionEventCollector) Store(e wire.Event, hash string) int {
	eventList := rec[hash]
	if rec.Contains(e, hash) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 50)
	}

	// store the event for the appropriate hash
	eventList = append(eventList, e)
	rec[hash] = eventList
	return len(eventList)
}
