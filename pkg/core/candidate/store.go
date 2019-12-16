package candidate

import (
	"bytes"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-wallet/block"
)

type (
	store struct {
		lock     sync.RWMutex
		messages map[string]*Candidate
	}

	Candidate struct {
		*block.Block
		*block.Certificate
	}
)

func NewCandidate() *Candidate {
	return &Candidate{block.NewBlock(), block.EmptyCertificate()}
}

func newStore() *store {
	return &store{
		messages: make(map[string]*Candidate),
	}
}

func (c *store) storeCandidateMessage(cm Candidate) error {
	// TODO: ensure we can't become a victim of memory overflow attacks

	// Make sure the hash is correct, to avoid malicious nodes from
	// overwriting the candidate block for a specific hash
	hash := cm.Block.Header.Hash
	if err := c.checkHash(hash, cm.Block); err != nil {
		return err
	}

	c.lock.Lock()
	c.messages[string(hash)] = &cm
	c.lock.Unlock()
	return nil
}

func (c *store) fetchCandidateMessage(hash []byte) *Candidate {
	c.lock.RLock()
	cm := c.messages[string(hash)]
	c.lock.RUnlock()
	return cm
}

func (c *store) checkHash(hash []byte, blk *block.Block) error {
	if err := blk.SetHash(); err != nil {
		return err
	}

	if !bytes.Equal(hash, blk.Header.Hash) {
		return errors.New("invalid block hash")
	}

	return nil
}

// Clear removes all candidate messages from or before a given round.
// Returns the amount of messages deleted.
func (c *store) Clear(round uint64) int {
	deletedCount := 0
	for h, m := range c.messages {
		if m.Block.Header.Height <= round {
			delete(c.messages, h)
			deletedCount++
		}
	}

	return deletedCount
}

func Decode(b *bytes.Buffer, cMsg *Candidate) error {
	if err := marshalling.UnmarshalBlock(b, cMsg.Block); err != nil {
		return err
	}

	return marshalling.UnmarshalCertificate(b, cMsg.Certificate)
}

func Encode(b *bytes.Buffer, cm *Candidate) error {
	if err := marshalling.MarshalBlock(b, cm.Block); err != nil {
		return err
	}

	return marshalling.MarshalCertificate(b, cm.Certificate)
}
