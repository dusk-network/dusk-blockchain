package testing

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// ConsensusRegistry holds all consensus-related data structures
// It should provide concurrency-safe accessors
//nolint:unused
type mockSafeRegistry struct {
	// lock protection per registry instance
	// Option: a mutex instance per a member
	lock sync.RWMutex

	p               user.Provisioners
	lastCertificate *block.Certificate
	lastCommittee   [][]byte
	chainTip        block.Block
}

//nolint:unused
func newMockSafeRegistry(genesis block.Block, cert *block.Certificate) *mockSafeRegistry {
	return &mockSafeRegistry{
		chainTip:        genesis,
		lastCertificate: cert,
	}
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

func (r *mockSafeRegistry) SetProvisioners(p user.Provisioners) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.p = p.Copy()
}

func (r *mockSafeRegistry) SetChainTip(b block.Block) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.chainTip = b.Copy().(block.Block)
}
