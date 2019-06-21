package chainsync

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type Counter struct {
	lock            sync.RWMutex
	blocksRemaining uint64
}

func NewCounter(subscriber wire.EventSubscriber) *Counter {
	sc := &Counter{}
	subscriber.SubscribeCallback(string(topics.AcceptedBlock), sc.decrement)
	return sc
}

func (s *Counter) decrement(b *bytes.Buffer) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.blocksRemaining > 0 {
		s.blocksRemaining--
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
	if heightDiff > 500 {
		heightDiff = 500
	}

	s.blocksRemaining = heightDiff
}
