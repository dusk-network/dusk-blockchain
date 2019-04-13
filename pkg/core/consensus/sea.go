package consensus

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// StepEventAccumulator is a helper struct for common operations on stored Event Arrays
// StepEventAccumulator is an helper for common operations on stored Event Arrays
type StepEventAccumulator struct {
	sync.RWMutex
	Map map[string][]wire.Event
}

func NewStepEventAccumulator() *StepEventAccumulator {
	return &StepEventAccumulator{
		Map: make(map[string][]wire.Event),
	}
}

// Clear up the Collector
func (sec *StepEventAccumulator) Clear() {
	sec.Lock()
	defer sec.Unlock()
	for key := range sec.Map {
		delete(sec.Map, key)
	}
}

// Contains checks if we already collected this event
func (sec *StepEventAccumulator) Contains(event wire.Event, step string) bool {
	sec.RLock()
	defer sec.RUnlock()
	for _, stored := range sec.Map[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (sec *StepEventAccumulator) Store(event wire.Event, step string) int {
	sec.RLock()
	eventList := sec.Map[step]
	sec.RUnlock()
	if sec.Contains(event, step) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	sec.Lock()
	sec.Map[step] = eventList
	sec.Unlock()
	return len(eventList)
}
