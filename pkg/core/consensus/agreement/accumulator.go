package agreement

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	log "github.com/sirupsen/logrus"
)

// Accumulator is an event accumulator, that will accumulate events until it
// reaches a certain threshold.
type Accumulator struct {
	handler            Handler
	verificationChan   chan message.Agreement
	eventChan          chan message.Agreement
	CollectedVotesChan chan []message.Agreement
	store              *store
}

// NewAccumulator initializes a worker pool, starts up an Accumulator and returns it.
func newAccumulator(handler Handler, workerAmount int) *Accumulator {
	// create accumulator
	a := &Accumulator{
		handler:            handler,
		verificationChan:   make(chan message.Agreement, 100),
		eventChan:          make(chan message.Agreement, 100),
		CollectedVotesChan: make(chan []message.Agreement, 1),
		store:              newStore(),
	}

	a.CreateWorkers(workerAmount)
	go a.Accumulate()
	return a
}

// Process a received Event, by passing it to a worker in the worker pool (if the event
// sender is part of the voting committee).
func (a *Accumulator) Process(ev message.Agreement) {
	defer func() {
		// we recover from panic in case of a late Process call which would attempt to write to the closed verificationChan
		// the alternative would be to never close the verificationChan and either use a multitude of channels to  stop the workers or a shared boolean set in the Accumulator.Stop
		if r := recover(); r != nil {
			lg.Traceln("attempting to forward to a closed Accumulator")
		}
	}()

	a.verificationChan <- ev
}

// Accumulate agreements per block hash until a quorum is reached or a stop is detected (by closing the internal event channel). Supposed to run in a goroutine
func (a *Accumulator) Accumulate() {
	for ev := range a.eventChan {
		hdr := ev.State()
		collected := a.store.Get(hdr.Step)
		weight := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)
		count := a.store.Insert(ev, weight)
		if count == len(collected) {
			lg.Warnln("Agreement was not accumulated since it is a duplicate")
			continue
		}

		lg.WithFields(log.Fields{
			"count":  count,
			"quorum": a.handler.Quorum(hdr.Round),
		}).Debugln("collected agreement")
		if count >= a.handler.Quorum(hdr.Round) {
			votes := a.store.Get(hdr.Step)
			a.CollectedVotesChan <- votes
			return
		}
	}
}

func (a *Accumulator) CreateWorkers(amount int) {
	var wg sync.WaitGroup

	if amount == 0 {
		amount = 4
	}

	wg.Add(amount)
	for i := 0; i < amount; i++ {
		go verify(a.verificationChan, a.eventChan, a.handler.Verify, &wg)
	}

	go func() {
		wg.Wait()
		close(a.eventChan)
	}()
}

func verify(verificationChan <-chan message.Agreement, eventChan chan<- message.Agreement, verifyFunc func(message.Agreement) error, wg *sync.WaitGroup) {
	for ev := range verificationChan {

		if err := verifyFunc(ev); err != nil {
			lg.WithError(err).Errorln("event verification failed")
			continue
		}

		select {
		case eventChan <- ev:
		default:
			lg.Warnln("accumulator skipped sending event")
		}
	}
	wg.Done()
}

// Stop kills the thread pool and shuts down the Accumulator.
func (a *Accumulator) Stop() {
	close(a.verificationChan)
}
