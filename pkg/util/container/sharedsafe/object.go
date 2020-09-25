package sharedsafe

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// Object implements copy-on-write technique based on payload.Safe
// Useful on sharing safely a Copyable data between goroutines
type Object struct {
	data payload.Safe
	mu   sync.RWMutex
}

// Set updates last block with a full copy of the new block
func (s *Object) Set(n payload.Safe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n != nil {
		s.data = n.Copy()
	}
}

// Get returns a safe copy of last block
func (s *Object) Get() payload.Safe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return nil
	}

	return s.data.Copy()
}
