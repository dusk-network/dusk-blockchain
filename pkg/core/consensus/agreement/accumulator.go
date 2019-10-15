package agreement

import (
	log "github.com/sirupsen/logrus"
)

// Accumulator is an event accumulator, that will accumulate events until it
// reaches a certain threshold.
type Accumulator struct {
	handler            Handler
	verificationChan   chan Agreement
	eventChan          chan Agreement
	CollectedVotesChan chan []Agreement
	store              *store
}

// NewAccumulator initializes a worker pool, starts up an Accumulator and returns it.
func newAccumulator(handler Handler, workerAmount int) *Accumulator {
	// set up worker pool
	eventChan := make(chan Agreement, 100)
	verificationChan := make(chan Agreement, 100)

	// create accumulator
	a := &Accumulator{
		handler:            handler,
		verificationChan:   verificationChan,
		eventChan:          eventChan,
		CollectedVotesChan: make(chan []Agreement, 1),
		store:              newStore(),
	}

	a.CreateWorkers(workerAmount)
	return a
}

// Process a received Event, by passing it to a worker in the worker pool (if the event
// sender is part of the voting committee).
func (a *Accumulator) Process(ev Agreement) {
	a.verificationChan <- ev
}

func (a *Accumulator) Accumulate() {
	for {
		ev, ok := <-a.eventChan
		if !ok {
			return
		}

		hash := string(ev.Header.BlockHash)
		count := a.store.Insert(ev, hash)
		if count >= a.handler.Quorum() {
			votes := a.store.Get(hash)
			a.CollectedVotesChan <- votes
			return
		}
	}
}

func (a *Accumulator) CreateWorkers(amount int) {
	if amount == 0 {
		amount = 4
	}

	for i := 0; i < amount; i++ {
		go verify(a.verificationChan, a.eventChan, a.handler.Verify)
	}
}

func verify(verificationChan <-chan Agreement, eventChan chan<- Agreement, verifyFunc func(Agreement) error) {
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
