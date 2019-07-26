package chainsync

import (
	"bytes"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var syncTime = 30 * time.Second

// Counter is a simple guarded counter, which can be used to figure out if we are currently
// syncing with another peer. It is used to orchestrate requests for blocks.
type Counter struct {
	lock            sync.RWMutex
	blocksRemaining uint64

	timer    *time.Timer
	stopChan chan struct{}
}

// NewCounter returns an initialized counter. It will decrement each time we accept a new block.
func NewCounter(subscriber wire.EventSubscriber) *Counter {
	sc := &Counter{stopChan: make(chan struct{})}
	subscriber.SubscribeCallback(string(topics.AcceptedBlock), sc.decrement)
	return sc
}

func (s *Counter) decrement(b *bytes.Buffer) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.blocksRemaining > 0 {
		s.blocksRemaining--

		// Stop the timer goroutine if we're done
		if s.blocksRemaining == 0 {
			s.stopChan <- struct{}{}
			return nil
		}

		// Refresh the timer whenever we get a new block during sync
		s.timer.Reset(syncTime)
	}

	return nil
}

func (s *Counter) isSyncing() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.blocksRemaining > 0
}

func (s *Counter) startSyncing(heightDiff uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// We can only receive up to 500 blocks at a time
	if heightDiff > 500 {
		heightDiff = 500
	}

	s.blocksRemaining = heightDiff
	s.timer = time.NewTimer(syncTime)
	go s.listenForTimer(s.timer)
}

func (s *Counter) listenForTimer(timer *time.Timer) {
	select {
	case <-timer.C:
		s.lock.Lock()
		defer s.lock.Unlock()
		s.blocksRemaining = 0
	case <-s.stopChan:
	}
}
