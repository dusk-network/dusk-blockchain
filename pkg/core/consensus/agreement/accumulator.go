package agreement

import (
	"bytes"
	"encoding/hex"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	log "github.com/sirupsen/logrus"
)

// AccumulatorHandler is a generic event handler with some added functionality, that is
// specific to the accumulator.
type AccumulatorHandler interface {
	EventHandler
	ExtractIdentifier(wire.Event, *bytes.Buffer) error
	Quorum() int
}

// Accumulator is an event accumulator, that will accumulate events until it
// reaches a certain threshold.
type Accumulator struct {
	wire.Store
	handler            AccumulatorHandler
	verificationChan   chan wire.Event
	eventChan          chan wire.Event
	CollectedVotesChan chan []wire.Event
}

// NewAccumulator initializes a worker pool, starts up an Accumulator and returns it.
func newAccumulator(handler AccumulatorHandler, store wire.Store) *Accumulator {
	// set up worker pool
	eventChan := make(chan wire.Event, 100)
	verificationChan := make(chan wire.Event, 100)

	// create accumulator
	a := &Accumulator{
		Store:              store,
		handler:            handler,
		verificationChan:   verificationChan,
		eventChan:          eventChan,
		CollectedVotesChan: make(chan []wire.Event, 1),
	}

	a.CreateWorkers()
	return a
}

// Process a received Event, by passing it to a worker in the worker pool (if the event
// sender is part of the voting committee).
func (a *Accumulator) Process(ev wire.Event) {
	a.verificationChan <- ev
}

func (a *Accumulator) Accumulate() {
	for {
		ev, ok := <-a.eventChan
		if !ok {
			return
		}

		b := new(bytes.Buffer)
		if err := a.handler.ExtractIdentifier(ev, b); err == nil {
			hash := hex.EncodeToString(b.Bytes())
			count := a.Insert(ev, hash)
			if count >= a.handler.Quorum() {
				votes := a.Get(hash)
				a.CollectedVotesChan <- votes
				return
			}
		}
	}
}

func (a *Accumulator) CreateWorkers() {
	amount := cfg.Get().Performance.AccumulatorWorkers
	if amount == 0 {
		amount = 4
	}

	for i := 0; i < amount; i++ {
		go verify(a.verificationChan, a.eventChan, a.handler.Verify)
	}
}

func verify(verificationChan <-chan wire.Event, eventChan chan<- wire.Event, verifyFunc func(wire.Event) error) {
	for {
		ev, ok := <-verificationChan
		if !ok {
			return
		}

		if err := verifyFunc(ev); err != nil {
			log.WithError(err).Errorln("event verification failed")
			continue
		}

		select {
		case eventChan <- ev:
		default:
			log.WithField("process", "accumulator worker").Debugln("skipped sending event")
		}
	}
}

// Stop kills the thread pool and shuts down the Accumulator.
func (a *Accumulator) Stop() {
	close(a.verificationChan)
	close(a.eventChan)
}
