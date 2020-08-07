package chainsync

import (
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
func NewCounter(subscriber eventbus.Subscriber) *Counter {
	sc := &Counter{stopChan: make(chan struct{})}
	decrementListener := eventbus.NewCallbackListener(sc.decrement)
	if config.Get().General.SafeCallbackListener {
		decrementListener = eventbus.NewSafeCallbackListener(sc.decrement)
	}
	subscriber.Subscribe(topics.AcceptedBlock, decrementListener)
	return sc
}

func (s *Counter) decrement(m message.Message) error {
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

// IsSyncing notifies whether the counter is syncing
func (s *Counter) IsSyncing() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.blocksRemaining > 0
}

// StartSyncing with the peers
func (s *Counter) StartSyncing(heightDiff uint64) {
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
