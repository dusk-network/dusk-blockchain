package consensus

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

// StepEventCollector is a helper struct for common operations on stored Event Arrays
type StepEventCollector map[string][]wire.Event

// Clear clears the map
func (sec StepEventCollector) Clear() {
	for key := range sec {
		delete(sec, key)
	}
}

// Contains checks if we have already collected this event for the specified step
func (sec StepEventCollector) Contains(event wire.Event, step string) bool {
	for _, stored := range sec[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store stores an event for a specific step
// Storing the same event for the same step, will be ignored.
// Returns the number of events stored at the specified step after the store operation
func (sec StepEventCollector) Store(event wire.Event, step string) int {
	eventList := sec[step]

	if sec.Contains(event, step) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 100)
	}

	eventList = append(eventList, event)
	sec[step] = eventList

	return len(eventList)
}

func (sec StepEventCollector) findEvents(step string) []wire.Event {
	eventList := sec[step]
	return eventList
}
