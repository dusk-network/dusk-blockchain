package consensus

import (
	"bytes"
	"sync"

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
		queue       *EventQueue
		handler     AccumulatorHandler
		state       State
		lock        sync.Mutex
		Accumulator *Accumulator
		checkStep   bool // in some cases, we do not check the step for relevancy
	}
)

// NewEventFilter returns an initialized EventFilter.
func NewEventFilter(handler AccumulatorHandler, state State, checkStep bool) *EventFilter {
	return &EventFilter{
		queue:     NewEventQueue(),
		handler:   handler,
		state:     state,
		checkStep: checkStep,
	}
}

// Collect an event buffer, deserialize it, and then pass it to the proper component.
func (ef *EventFilter) Collect(buffer *bytes.Buffer) error {
	ef.lock.Lock()
	ev, err := ef.handler.Deserialize(buffer)
	if err != nil {
		ef.lock.Unlock()
		return err
	}

	header := ef.handler.ExtractHeader(ev)
	roundDiff, stepDiff := ef.state.Cmp(header.Round, header.Step)
	if ef.isEarly(roundDiff, stepDiff) {
		ef.queue.PutEvent(header.Round, header.Step, ev)
		ef.lock.Unlock()
		return nil
	}

	if ef.isRelevant(roundDiff, stepDiff) {
		ef.Accumulator.Process(ev)
	}

	ef.lock.Unlock()
	return nil
}

func (ef *EventFilter) isEarly(roundDiff, stepDiff int) bool {
	earlyRound := roundDiff < 0
	if !ef.checkStep {
		return earlyRound
	}
	earlyStep := stepDiff < 0
	sameRound := roundDiff == 0
	return earlyRound || (sameRound && earlyStep)
}

func (ef *EventFilter) isRelevant(roundDiff, stepDiff int) bool {
	relevantRound := roundDiff == 0
	if !ef.checkStep {
		return relevantRound
	}
	relevantStep := stepDiff == 0
	return relevantRound && relevantStep
}

// UpdateRound updates the state for the EventFilter, and empties the queue of
// obsolete events.
func (ef *EventFilter) UpdateRound(round uint64) {
	ef.lock.Lock()
	defer ef.lock.Unlock()
	ef.state.Update(round)
	ef.Accumulator = NewAccumulator(ef.handler, NewAccumulatorStore(), ef.state, ef.checkStep)
	ef.Accumulator.CreateWorkers()
	go ef.Accumulator.Accumulate()
	ef.queue.Clear(round - 1)
}

func (ef *EventFilter) ResetAccumulator() {
	ef.lock.Lock()
	defer ef.lock.Unlock()
	ef.Accumulator = NewAccumulator(ef.handler, NewAccumulatorStore(), ef.state, ef.checkStep)
	ef.Accumulator.CreateWorkers()
	go ef.Accumulator.Accumulate()
}

// FlushQueue will retrieve all queued events for a certain point in consensus,
// and hand them off to the Processor.
func (ef *EventFilter) FlushQueue() {
	var queuedEvents []wire.Event
	if ef.checkStep {
		queuedEvents = ef.queue.GetEvents(ef.state.Round(), ef.state.Step())
	} else {
		queuedEvents = ef.queue.Flush(ef.state.Round())
	}

	ef.lock.Lock()
	defer ef.lock.Unlock()
	for _, event := range queuedEvents {
		ef.Accumulator.Process(event)
	}
}
