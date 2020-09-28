package chainsync

import (
	"sync/atomic"
	"time"
)

var syncTime = 30 * time.Second

// Counter is a simple guarded counter, which can be used to figure out if we are currently
// syncing with another peer. It is used to orchestrate requests for blocks.
type Counter struct {
	blocksRemaining uint64

	timer    *time.Timer
	stopChan chan struct{}
}

// NewCounter counts remaining blocks when synchronizing. It is called through
// an RPCBus call
func NewCounter() *Counter {
	sc := &Counter{
		stopChan: make(chan struct{}),
	}
	atomic.StoreUint64(&sc.blocksRemaining, 0)
	return sc
}

//Decrement the number of remaining blocks until the node is fully synchronized
func (s *Counter) Decrement() {
	if atomic.LoadUint64(&s.blocksRemaining) > 0 {
		atomic.AddUint64(&s.blocksRemaining, ^uint64(0))

		// Stop the timer goroutine if we're done
		if atomic.LoadUint64(&s.blocksRemaining) == 0 {
			s.stopChan <- struct{}{}
			return
		}

		// Refresh the timer whenever we get a new block during sync
		s.timer.Reset(syncTime)
	}
}

// IsSyncing notifies whether the counter is syncing
func (s *Counter) IsSyncing() bool {
	return atomic.LoadUint64(&s.blocksRemaining) > 0
}

// StartSyncing with the peers
func (s *Counter) StartSyncing(heightDiff uint64) {
	// We can only receive up to 500 blocks at a time
	if heightDiff > 500 {
		heightDiff = 500
	}

	atomic.StoreUint64(&s.blocksRemaining, heightDiff)
	s.timer = time.NewTimer(syncTime)
	go s.listenForTimer(s.timer)
}

func (s *Counter) listenForTimer(timer *time.Timer) {
	select {
	case <-timer.C:
		atomic.StoreUint64(&s.blocksRemaining, 0)
	case <-s.stopChan:
	}
}
