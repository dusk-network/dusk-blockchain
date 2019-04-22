package consensus

import (
	"bytes"
	"encoding/hex"
	"errors"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// AccumulatorHandler is a generic event handler with some added functionality, that is
// specific to the accumulator.
type AccumulatorHandler interface {
	EventHandler
	committee.Committee
	ExtractIdentifier(wire.Event, *bytes.Buffer) error
}

// Accumulator is a generic event accumulator, that will accumulate events until it
// reaches a certain threshold.
type Accumulator struct {
	*AccumulatorStore
	handler            AccumulatorHandler
	CollectedVotesChan chan []wire.Event
}

func NewAccumulator(handler AccumulatorHandler) *Accumulator {
	return &Accumulator{
		AccumulatorStore:   NewAccumulatorStore(),
		handler:            handler,
		CollectedVotesChan: make(chan []wire.Event, 1),
	}
}

func (a *Accumulator) Process(ev wire.Event) {
	if a.shouldSkip(ev) {
		log.WithError(errors.New("sender not part of committee")).Debugln("event dropped")
		return
	}

	if err := a.handler.Verify(ev); err != nil {
		log.WithError(err).Debugln("Voteset verification failed")
		return
	}
	a.accumulate(ev)
}

func (a *Accumulator) accumulate(ev wire.Event) {
	b := new(bytes.Buffer)
	if err := a.handler.ExtractIdentifier(ev, b); err == nil {
		hash := hex.EncodeToString(b.Bytes())
		count := a.Store(ev, hash)
		if count == a.handler.Quorum() {
			votes := a.Get(hash)
			a.CollectedVotesChan <- votes
			a.Clear()
		}
	}
}

func (a *Accumulator) GetAllEvents() []wire.Event {
	allEvents := make([]wire.Event, 0)
	a.RLock()
	defer a.RUnlock()
	for _, evs := range a.Map {
		allEvents = append(allEvents, evs...)
	}

	return allEvents
}

// ShouldSkip checks if the message is propagated by a committee member.
func (a *Accumulator) shouldSkip(ev wire.Event) bool {
	header := a.handler.ExtractHeader(ev)
	return !a.handler.IsMember(ev.Sender(), header.Round, header.Step)
}
