package chainsync

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var syncTime = 30 * time.Second

// Counter is a simple guarded counter, which can be used to figure out if we are currently
// syncing with another peer. It is used to orchestrate requests for blocks.
type Counter struct {
	lock            sync.RWMutex
	blocksRemaining uint64
	bus             *rpcbus.RPCBus

	timer                *time.Timer
	stopChan             chan struct{}
	getAcceptedBlockChan <-chan rpcbus.Request
}

// NewCounter counts remaining blocks when synchronizing. It is called through
// an RPCBus call
// TODO: rename the RPC topic to DECREMENT
func NewCounter(bus *rpcbus.RPCBus) (*Counter, error) {
	getAcceptedBlockChan := make(chan rpcbus.Request)
	if err := bus.Register(topics.AcceptedBlock, getAcceptedBlockChan); err != nil {
		return nil, err
	}

	sc := &Counter{
		bus:                  bus,
		stopChan:             make(chan struct{}),
		getAcceptedBlockChan: getAcceptedBlockChan,
	}

	go sc.listen()
	return sc, nil
}

func (s *Counter) listen() {
	for {
		select {
		case req := <-s.getAcceptedBlockChan:
			s.decrement()
			req.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}, Err: nil}
		case <-s.stopChan:
			return
		}
	}
}

func (s *Counter) decrement() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.blocksRemaining > 0 {
		s.blocksRemaining--

		// Stop the timer goroutine if we're done
		if s.blocksRemaining == 0 {
			s.stopChan <- struct{}{}
			return
		}

		// Refresh the timer whenever we get a new block during sync
		s.timer.Reset(syncTime)
	}
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
