package candidate

import (
	"errors"
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
	defer c.lock.Unlock()
	c.messages[string(cm.Block.Header.Hash)] = cm
}

// fetchCandidateMessage returns a deep copy of the Candidate of this hash or an error if not found.
func (c *store) fetchCandidateMessage(hash []byte) (message.Candidate, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	cm, ok := c.messages[string(hash)]

	if ok {
		return cm.Copy().(message.Candidate), nil
	}
	return cm, errors.New("not found")
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
