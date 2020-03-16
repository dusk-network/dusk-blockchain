package candidate

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

type (
	store struct {
		lock     sync.RWMutex
		messages map[string]message.Candidate
	}
)

func newStore() *store {
	return &store{
		messages: make(map[string]message.Candidate),
	}
}

func (c *store) storeCandidateMessage(cm message.Candidate) {
	// TODO: ensure we can't become a victim of memory overflow attacks
	c.lock.Lock()
	c.messages[string(cm.Block.Header.Hash)] = cm
	c.lock.Unlock()
}

func (c *store) fetchCandidateMessage(hash []byte) message.Candidate {
	c.lock.RLock()
	cm := c.messages[string(hash)]
	c.lock.RUnlock()
	return cm
}

// Clear removes all candidate messages from or before a given round.
// Returns the amount of messages deleted.
func (c *store) Clear(round uint64) int {
	c.lock.Lock()
	defer c.lock.Unlock()
	deletedCount := 0
	for h, m := range c.messages {
		if m.Block.Header.Height <= round {
			delete(c.messages, h)
			deletedCount++
		}
	}

	return deletedCount
}
