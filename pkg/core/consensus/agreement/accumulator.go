// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"sync"
	"time"

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
	storeMap           *storeMap

	workersQuitChan chan struct{}
}

// NewAccumulator initializes a worker pool, starts up an Accumulator and returns it.
func newAccumulator(handler Handler, workerAmount int) *Accumulator {
	// create accumulator
	a := &Accumulator{
		handler:            handler,
		verificationChan:   make(chan message.Agreement, 100),
		eventChan:          make(chan message.Agreement, 100),
		CollectedVotesChan: make(chan []message.Agreement, 1),
		storeMap:           newStoreMap(),
		workersQuitChan:    make(chan struct{}),
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
		// the alternative would be to never close the verificationChan and either use a multitude of channels to stop the workers or a shared boolean set in the Accumulator.Stop
		if r := recover(); r != nil {
			lg.Traceln("attempting to forward to a closed Accumulator")
		}
	}()

	a.verificationChan <- ev
}

// Accumulate agreements per block hash until a quorum is reached or a stop is detected (by closing the internal event channel). Supposed to run in a goroutine.
func (a *Accumulator) Accumulate() {
	for ev := range a.eventChan {
		hdr := ev.State()

		// Obtain corresponding block agreement cache given its hash
		var s *store
		if s = a.storeMap.getStoreByHash(hdr.BlockHash); s == nil {
			s = a.storeMap.makeStoreByHash(hdr.BlockHash)
		}

		// Try to add agreement to our cache
		collected := s.Get(hdr.Step)
		weight := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)

		count := s.Insert(ev, weight)
		if count == len(collected) {
			lg.Warnln("Agreement was not accumulated since it is a duplicate")
			continue
		}

		lg.WithFields(log.Fields{
			"step":       ev.State().Step,
			"round":      ev.State().Round,
			"aggr_count": count,
			"quorum":     a.handler.Quorum(hdr.Round),
			"hash_count": a.storeMap.len(),
		}).Debug("collected agreement")

		if count >= a.handler.Quorum(hdr.Round) {
			votes := s.Get(hdr.Step)
			a.CollectedVotesChan <- votes

			lg.WithFields(log.Fields{
				"step":        ev.State().Step,
				"round":       ev.State().Round,
				"aggr_count":  count,
				"quorum":      a.handler.Quorum(hdr.Round),
				"hash_count":  a.storeMap.len(),
				"steps_count": s.Len(),
				"duration":    time.Now().Unix() - s.CreatedAt(),
			}).Info("quorum reached")

			return
		}
	}
}

// CreateWorkers creates an amount of workers that verify Agreement messages
// concurrently.
func (a *Accumulator) CreateWorkers(amount int) {
	var wg sync.WaitGroup

	if amount == 0 {
		amount = 4
	}

	wg.Add(amount)

	for i := 0; i < amount; i++ {
		go verify(a.verificationChan, a.eventChan, a.handler.Verify, &wg, a.workersQuitChan)
	}

	go func() {
		wg.Wait()
		close(a.eventChan)
	}()
}

func verify(verificationChan <-chan message.Agreement, eventChan chan<- message.Agreement, verifyFunc func(message.Agreement) error, wg *sync.WaitGroup, quit chan struct{}) {
	defer wg.Done()

	for {
		select {
		case ev := <-verificationChan:
			if err := verifyFunc(ev); err != nil {
				lg.WithError(err).Errorln("event verification failed")
				break
			}

			select {
			case eventChan <- ev:
			default:
				lg.Warnln("accumulator skipped sending event")
			}
		case <-quit:
			return
		}
	}
}

// Stop kills the thread pool and shuts down the Accumulator.
func (a *Accumulator) Stop() {
	close(a.workersQuitChan)
}
