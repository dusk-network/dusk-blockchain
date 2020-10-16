package sharedsafe

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// Object implements copy-on-write technique based on payload.Safe
// Useful on sharing safely a Copyable data between goroutines
// Multiple readers, single writer at a time
type Object struct {
	data payload.Safe
	mu   sync.RWMutex
}

// Set updates inner value with a deep copy
// If n is nil, inner value is reset
// concurrency safe
func (s *Object) Set(n payload.Safe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n != nil {
		s.data = n.Copy()
	} else {
		// reset payload
		n = nil
	}
}

// Get returns a deep copy of the inner value
// concurrency safe
func (s *Object) Get() payload.Safe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return nil
	}

	return s.data.Copy()
}
