package consensus

import (
	"bytes"
	"encoding/hex"
	"errors"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
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
	wire.Store
	handler            AccumulatorHandler
	verificationChan   chan<- wire.Event
	eventChan          <-chan wire.Event
	CollectedVotesChan chan []wire.Event
}

// NewAccumulator initializes a worker pool, starts up an Accumulator and returns it.
func NewAccumulator(handler AccumulatorHandler, store wire.Store) *Accumulator {
	// set up worker pool
	eventChan := make(chan wire.Event, 10)
	verificationChan := createWorkers(eventChan, handler.Verify)

	// create accumulator
	a := &Accumulator{
		Store:              store,
		handler:            handler,
		verificationChan:   verificationChan,
		eventChan:          eventChan,
		CollectedVotesChan: make(chan []wire.Event, 1),
	}

	go a.accumulate()
	return a
}

// Process a received Event, by passing it to a worker in the worker pool (if the event
// sender is part of the voting committee).
func (a *Accumulator) Process(ev wire.Event) {
	if a.shouldSkip(ev) {
		log.WithError(errors.New("sender not part of committee")).Debugln("event dropped")
		return
	}

	a.verificationChan <- ev
}

func (a *Accumulator) accumulate() {
	for {
		ev := <-a.eventChan
		b := new(bytes.Buffer)
		if err := a.handler.ExtractIdentifier(ev, b); err == nil {
			hash := hex.EncodeToString(b.Bytes())
			count := a.Insert(ev, hash)
			if count >= a.handler.Quorum() {
				votes := a.Get(hash)
				a.CollectedVotesChan <- votes
				a.Clear()
			}
		}
	}
}

// ShouldSkip checks if the message is propagated by a committee member.
func (a *Accumulator) shouldSkip(ev wire.Event) bool {
	header := a.handler.ExtractHeader(ev)
	return !a.handler.IsMember(ev.Sender(), header.Round, header.Step)
}

func createWorkers(eventChan chan<- wire.Event, verifyFunc func(wire.Event) error) chan<- wire.Event {
	verificationChan := make(chan wire.Event, 100)
	amount := cfg.Get().Performance.AccumulatorWorkers
	if amount == 0 {
		amount = 4
	}

	for i := 0; i < amount; i++ {
		go verify(verificationChan, eventChan, verifyFunc)
	}

	return verificationChan
}

func verify(verificationChan <-chan wire.Event, eventChan chan<- wire.Event, verifyFunc func(wire.Event) error) {
	for {
		ev := <-verificationChan
		if err := verifyFunc(ev); err != nil {
			continue
		}

		eventChan <- ev
	}
}
