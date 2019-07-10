package selection

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

type filter struct {
	queue    *consensus.EventQueue
	handler  ScoreEventHandler
	state    consensus.State
	selector *eventSelector
}

func newFilter(handler ScoreEventHandler, state consensus.State, selector *eventSelector) *filter {
	return &filter{
		queue:    consensus.NewEventQueue(),
		handler:  handler,
		state:    state,
		selector: selector,
	}
}

func (f *filter) Collect(buffer *bytes.Buffer) error {
	ev, err := f.handler.Deserialize(buffer)
	if err != nil {
		return err
	}

	header := f.handler.ExtractHeader(ev)
	roundDiff, stepDiff := f.state.Cmp(header.Round, header.Step)
	if f.isEarly(roundDiff, stepDiff) {
		f.queue.PutEvent(header.Round, header.Step, ev)
		return nil
	}

	if f.isRelevant(roundDiff, stepDiff) {
		f.selector.Process(ev)
	}

	return nil
}

func (f *filter) isEarly(roundDiff, stepDiff int) bool {
	earlyRound := roundDiff < 0
	return earlyRound
}

func (f *filter) isRelevant(roundDiff, stepDiff int) bool {
	relevantRound := roundDiff == 0
	return relevantRound
}

func (f *filter) UpdateRound(round uint64) {
	f.state.Update(round)
	f.queue.Clear(round - 1)
}

func (f *filter) FlushQueue() {
	queuedEvents := f.queue.Flush(f.state.Round())
	for _, ev := range queuedEvents {
		f.selector.Process(ev)
	}
}
