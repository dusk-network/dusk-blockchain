package chainsync

import (
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

var syncTime = 30 * time.Second

// maxBlocksCount maximum number of blocks to fetch from a single peer on syncing procedure
const maxBlocksCount = 500

// Counter is a simple guarded counter, which can be used to figure out if we are currently
// syncing with another peer. It is used to orchestrate requests for blocks.
// Counter also represents a shared state amongst all peer connection synchronizers
type Counter struct {
	lock sync.RWMutex

	// blocksRemaining remaining blocks to catch up with so-called syncing peer
	// blocksRemaining  > 0 has meaning of sync mode
	blocksRemaining uint64
	// syncingPeerAddr is the address of the peer we are syncing with (while in syncMode)
	// Only 1 syncingPeer at a time to prevent asking
	// many peers for the same blocks.
	syncingPeerAddr string

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

// decrement decrements blockRemaining if a new block is accepted by the chain
func (s *Counter) decrement(m message.Message) error {

	if !s.IsSyncing() {
		// no need to decrement in non-syncing mode
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.blocksRemaining > 0 {
		s.blocksRemaining--

		// Stop the timer goroutine if we're done
		if s.blocksRemaining == 0 {
			s.syncingPeerAddr = ""
			s.stopChan <- struct{}{}
			return nil
		}

		// Refresh the timer whenever we get a new block during sync
		// TODO: Reconsider syncTime default value. Currently the entire sync procedure
		// with a single peer could consume up to 25mins (maxBlocksCount*syncTime)
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

// StartSyncing with a single peer
func (s *Counter) StartSyncing(heightDiff uint64, syncPeerAddr string) (string, error) {

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.blocksRemaining > 0 {
		return s.syncingPeerAddr, errors.New("already syncing")
	}

	s.syncingPeerAddr = syncPeerAddr

	// We can only receive up to maxBlocksCount blocks at a time
	if heightDiff > maxBlocksCount {
		heightDiff = maxBlocksCount
	}

	s.blocksRemaining = heightDiff
	s.timer = time.NewTimer(syncTime)
	go s.listenForTimer(s.timer)

	return s.syncingPeerAddr, nil
}

func (s *Counter) listenForTimer(timer *time.Timer) {
	select {
	case <-timer.C:
		s.lock.Lock()
		defer s.lock.Unlock()
		s.blocksRemaining = 0
		s.syncingPeerAddr = ""
	case <-s.stopChan:
	}

}

// GetSyncingPeer returns address of the syncing Peer (IP:Port)
func (s *Counter) GetSyncingPeer() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.syncingPeerAddr
}
