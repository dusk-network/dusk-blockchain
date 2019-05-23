package consensus

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// EventHandler encapsulates logic specific to the various EventFilters.
	// Each EventFilter needs to verify, prioritize and extract information from Events.
	// EventHandler is the interface that abstracts these operations away.
	// The implementors of this interface is the real differentiator of the various
	// consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventMarshaller
		wire.EventDeserializer
		ExtractHeader(wire.Event) *header.Header
	}

	// EventFilter is a generic wire.Collector that can be used by consensus components
	// for filtering and passing down messages. It coordinates an EventQueue to manage
	// Events coming too early and delegates consensus specific logic to the handler.
	EventFilter struct {
		queue     *EventQueue
		handler   EventHandler
		state     State
		processor EventProcessor
		checkStep bool // in some cases, we do not check the step for relevancy
	}

	// EventProcessor is an abstraction over a process that receives events
	// from an EventFilter.
	EventProcessor interface {
		Process(wire.Event)
	}
)

// NewEventFilter returns an initialized EventFilter.
func NewEventFilter(handler EventHandler, state State, processor EventProcessor,
	checkStep bool) *EventFilter {
	return &EventFilter{
		queue:     NewEventQueue(),
		handler:   handler,
		state:     state,
		processor: processor,
		checkStep: checkStep,
	}
}

// Collect an event buffer, deserialize it, and then pass it to the proper component.
func (c *EventFilter) Collect(buffer *bytes.Buffer) error {
	ev, err := c.handler.Deserialize(buffer)
	if err != nil {
		return err
	}

	header := c.handler.ExtractHeader(ev)
	roundDiff, stepDiff := c.state.Cmp(header.Round, header.Step)
	if c.isEarly(roundDiff, stepDiff) {
		c.queue.PutEvent(header.Round, header.Step, ev)
		return nil
	}

	if c.isRelevant(roundDiff, stepDiff) {
		c.processor.Process(ev)
	}

	return nil
}

func (c *EventFilter) isEarly(roundDiff, stepDiff int) bool {
	earlyRound := roundDiff < 0
	if !c.checkStep {
		return earlyRound
	}
	earlyStep := stepDiff < 0
	sameRound := roundDiff == 0
	return earlyRound || (sameRound && earlyStep)
}

func (c *EventFilter) isRelevant(roundDiff, stepDiff int) bool {
	relevantRound := roundDiff == 0
	if !c.checkStep {
		return relevantRound
	}
	relevantStep := stepDiff == 0
	return relevantRound && relevantStep
}

// UpdateRound updates the state for the EventFilter, and empties the queue of
// obsolete events.
func (c *EventFilter) UpdateRound(round uint64) {
	c.state.Update(round)
	c.queue.Clear(round - 1)
}

// FlushQueue will retrieve all queued events for a certain point in consensus,
// and hand them off to the Processor.
func (c *EventFilter) FlushQueue() {
	var queuedEvents []wire.Event
	if c.checkStep {
		queuedEvents = c.queue.GetEvents(c.state.Round(), c.state.Step())
	} else {
		queuedEvents = c.queue.Flush(c.state.Round())
	}

	for _, event := range queuedEvents {
		c.processor.Process(event)
	}
}
