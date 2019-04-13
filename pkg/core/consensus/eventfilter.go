package consensus

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// EventHandler encapsulate logic specific to the various EventFilters.
	// Each EventFilter needs to verify, prioritize and extract information from Events.
	// EventHandler is the interface that abstracts these operations away.
	// The implementors of this interface is the real differentiator of the various
	// consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventUnMarshaller
		wire.SignatureMarshaller
		NewEvent() wire.Event
		ExtractHeader(wire.Event, *events.Header)
	}

	// EventFilter is a generic filter that can be used by consensus components for
	// filtering and passing down messages.
	EventFilter struct {
		committee committee.Committee
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

func NewEventFilter(publisher wire.EventPublisher, committee committee.Committee,
	handler EventHandler, state State, processor EventProcessor, checkStep bool) *EventFilter {
	return &EventFilter{
		committee: committee,
		queue:     NewEventQueue(),
		handler:   handler,
		state:     state,
		processor: processor,
		checkStep: checkStep,
	}
}

func (c *EventFilter) Collect(buffer *bytes.Buffer) error {
	ev := c.handler.NewEvent()
	if err := c.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	if c.shouldSkip(ev) {
		return errors.New("sender not part of committee")
	}

	header := &events.Header{}
	c.handler.ExtractHeader(ev, header)
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
	return earlyRound || earlyStep
}

func (c *EventFilter) isRelevant(roundDiff, stepDiff int) bool {
	relevantRound := roundDiff == 0
	if !c.checkStep {
		return relevantRound
	}
	relevantStep := stepDiff == 0
	return relevantRound && relevantStep
}

// ShouldSkip checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
func (c *EventFilter) shouldSkip(ev wire.Event) bool {
	return !c.committee.IsMember(ev.Sender())
}

func (c *EventFilter) UpdateRound(round uint64) {
	c.state.Update(round)
	c.queue.Clear(round - 1)
}

func (c *EventFilter) FlushQueue() {
	queuedEvents := c.queue.GetEvents(c.state.Round(), c.state.Step())
	for _, event := range queuedEvents {
		c.processor.Process(event)
	}
}
