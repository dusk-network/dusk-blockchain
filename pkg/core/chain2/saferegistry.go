package chain2

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// ConsensusRegistry holds all consensus-related data structures
// It should provide concurrency-safe accessors
//nolint:unused
type SafeRegistry struct {

	// lock protection per registry instance
	// Option: a mutex instance per a member
	lock sync.RWMutex

	p               user.Provisioners
	lastCertificate block.Certificate
	lastCommittee   [][]byte
	chainTip        block.Block
}

//nolint:unused
func newSafeRegistry(chainTip *block.Block, cert *block.Certificate, p *user.Provisioners) *SafeRegistry {

	return &SafeRegistry{
		chainTip:        chainTip.Copy().(block.Block),
		lastCertificate: *cert.Copy(),
		p:               p.Copy(),
	}
}

func (r *SafeRegistry) GetProvisioners() user.Provisioners {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.p.Copy()
}

func (r *SafeRegistry) GetChainTip() block.Block {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.chainTip.Copy().(block.Block)
}

func (r *SafeRegistry) GetLastCertificate() *block.Certificate {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.lastCertificate.Copy()
}

func (r *SafeRegistry) GetLastCommittee() [][]byte {
	r.lock.RLock()
	defer r.lock.RUnlock()

	dup := make([][]byte, len(r.lastCommittee))
	for i := range r.lastCommittee {
		dup[i] = make([]byte, len(r.lastCommittee[i]))
		copy(dup[i], r.lastCommittee[i])
	}

	return dup
}

func (r *SafeRegistry) SetProvisioners(p user.Provisioners) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.p = p.Copy()
}

func (r *SafeRegistry) SetChainTip(b block.Block) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.chainTip = b.Copy().(block.Block)
}
