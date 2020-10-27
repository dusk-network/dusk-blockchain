package testing

import (
	"bytes"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// ConsensusRegistry holds all consensus-related data structures
// It should provide concurrency-safe accessors
type mockSafeRegistry struct {

	// lock protection per registry instance
	// TODO: a mutex instance per a member
	lock sync.RWMutex

	p               *user.Provisioners
	lastCertificate *block.Certificate
	lastCommittee   [][]byte
	chainTip        block.Block
	candidates      []message.Candidate
}

func newMockSafeRegistry() *mockSafeRegistry {

	randomGenesis := block.NewBlock()

	return &mockSafeRegistry{
		chainTip: *randomGenesis,
	}
}

// RetrieveCandidate returns a copy of candidate block if found by hash
func (r *mockSafeRegistry) GetCandidateByHash(hash []byte) (block.Block, error) {

	r.lock.RLock()
	defer r.lock.RUnlock()

	if len(hash) != 32 {
		return block.Block{}, errors.New("invalid hash")
	}

	for n := 0; n < len(r.candidates); n++ {
		b := r.candidates[n].Block
		if bytes.Equal(b.Header.Hash, hash) {
			return b.Copy().(block.Block), nil
		}
	}

	return block.Block{}, errors.New("candidate not found")
}

func (r *mockSafeRegistry) GetProvisioners() user.Provisioners {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.p.Copy()
}

func (r *mockSafeRegistry) GetChainTip() block.Block {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.chainTip.Copy().(block.Block)
}

func (r *mockSafeRegistry) GetLastCertificate() *block.Certificate {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.lastCertificate.Copy()
}

func (r *mockSafeRegistry) GetLastCommittee() [][]byte {
	r.lock.RLock()
	defer r.lock.RUnlock()

	dup := make([][]byte, len(r.lastCommittee))
	for i := range r.lastCommittee {
		dup[i] = make([]byte, len(r.lastCommittee[i]))
		copy(dup[i], r.lastCommittee[i])
	}

	return dup
}

func (r *mockSafeRegistry) SetChainTip(b block.Block) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.chainTip = b.Copy().(block.Block)
}

func (r *mockSafeRegistry) AddCandidate(m message.Candidate) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.candidates) == 0 {
		r.candidates = make([]message.Candidate, 0)
	}
	r.candidates = append(r.candidates, m)
}
