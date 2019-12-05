package chain

import (
	"bytes"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
)

type (
	candidateStore struct {
		lock     sync.RWMutex
		messages map[string]*candidateMsg

		// Needed for publishing GetCandidate messages and receiving
		// Candidate messages in return.
		broker eventbus.Broker
	}

	candidateMsg struct {
		blk  *block.Block
		cert *block.Certificate
	}
)

func newCandidateStore(broker eventbus.Broker) *candidateStore {
	c := &candidateStore{
		messages: make(map[string]*candidateMsg),
		broker:   broker,
	}

	broker.Subscribe(topics.Candidate, eventbus.NewCallbackListener(c.storeCandidateMessage))
	broker.Register(topics.Candidate, consensus.NewRepublisher(broker, topics.Candidate), &consensus.Validator{})
	return c
}

func (c *candidateStore) storeCandidateMessage(b bytes.Buffer) error {
	cm, err := decodeCandidateMessage(b)
	if err != nil {
		return err
	}

	// TODO: ensure we can't become a victim of memory overflow attacks

	// Make sure the hash is correct, to avoid malicious nodes from
	// overwriting the candidate block for a specific hash
	hash := cm.blk.Header.Hash
	if err := c.checkHash(hash, cm.blk); err != nil {
		return err
	}

	c.lock.Lock()
	c.messages[string(hash)] = cm
	c.lock.Unlock()
	return nil
}

func (c *candidateStore) fetchCandidateMessage(hash []byte) (*candidateMsg, error) {
	c.lock.RLock()
	cm := c.messages[string(hash)]
	c.lock.RUnlock()

	if cm == nil {
		return c.requestCandidateMessage(hash)
	}

	return cm, nil
}

func (c *candidateStore) requestCandidateMessage(hash []byte) (*candidateMsg, error) {
	// Set up an extra listener
	candidateChan := make(chan bytes.Buffer, 100)
	id := c.broker.Subscribe(topics.Candidate, eventbus.NewChanListener(candidateChan))
	defer c.broker.Unsubscribe(topics.Candidate, id)

	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, hash); err != nil {
		return nil, err
	}

	if err := topics.Prepend(buf, topics.GetCandidate); err != nil {
		return nil, err
	}

	c.broker.Publish(topics.Gossip, buf)

	// Listen until we receive the requested candidate
	// NOTE: currently this could loop infinitely if nobody responds.
	for {
		b := <-candidateChan
		cm, err := decodeCandidateMessage(b)
		if err != nil {
			return nil, err
		}

		hash := cm.blk.Header.Hash
		if err := c.checkHash(hash, cm.blk); err != nil {
			return nil, err
		}

		return cm, nil
	}
}

func (c *candidateStore) checkHash(hash []byte, blk *block.Block) error {
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
func (c *candidateStore) Clear(round uint64) int {
	deletedCount := 0
	for h, m := range c.messages {
		if m.blk.Header.Height <= round {
			delete(c.messages, h)
			deletedCount++
		}
	}

	return deletedCount
}

func decodeCandidateMessage(b bytes.Buffer) (*candidateMsg, error) {
	blk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&b, blk); err != nil {
		return nil, err
	}

	cert := block.EmptyCertificate()
	if err := marshalling.UnmarshalCertificate(&b, cert); err != nil {
		return nil, err
	}

	return &candidateMsg{blk, cert}, nil
}

func encodeCandidateMessage(b *bytes.Buffer, cm *candidateMsg) error {
	if err := marshalling.MarshalBlock(b, cm.blk); err != nil {
		return err
	}

	return marshalling.MarshalCertificate(b, cm.cert)
}
