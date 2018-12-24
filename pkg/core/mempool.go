package core

import (
	"encoding/hex"
	"fmt"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// MemPool is a concurrent-safe mapping of tx hashes to their structs.
type MemPool struct {
	txs map[string]*transactions.Stealth

	lock *sync.RWMutex
}

// Init will initialize the tx map of the memory pool.
func (m *MemPool) Init() {
	m.txs = make(map[string]*transactions.Stealth)
}

// AddTx will spawn a goroutine to add a transaction to the mempool.
// A transaction should be verified by a Blockchain instance before
// being passed to the memory pool, so don't call it directly, but
// rather use AcceptTransaction.
// This function is safe for concurrent access.
func (m *MemPool) AddTx(tx *transactions.Stealth) {
	go m.addTx(tx)
}

// RemoveTx will spawn a goroutine to remove a transaction from the mempool.
// This function is safe for concurrent access.
func (m *MemPool) RemoveTx(tx *transactions.Stealth) {
	go m.removeTx(tx)
}

// GetTx will get a transaction from the mempool using the passed hash.
// This function is safe for concurrent access.
func (m *MemPool) GetTx(hash string) (*transactions.Stealth, error) {
	m.lock.RLock()
	tx, exists := m.txs[hash]
	m.lock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tx %v is not in the mempool", hash)
	}

	return tx, nil
}

// Exists checks by hash whether a transaction exists in the mempool.
// This function is safe for concurrent access.
func (m *MemPool) Exists(hash string) bool {
	m.lock.RLock()
	_, exists := m.txs[hash]
	m.lock.RUnlock()
	return exists
}

// Count returns the amount of items in the mempool.
// This function is safe for concurrent access.
func (m *MemPool) Count() int {
	m.lock.RLock()
	l := len(m.txs)
	m.lock.RUnlock()
	return l
}

// GetAllHashes will return all the tx hashes in the mempoool
// as hexadecimal strings.
// This function is safe for concurrent access.
func (m *MemPool) GetAllHashes() []string {
	m.lock.RLock()
	hashes := make([]string, len(m.txs))
	i := 0
	for hash := range m.txs {
		hashes[i] = hash
		i++
	}

	m.lock.RUnlock()
	return hashes
}

// GetAllTxs will return an array of all txs in the mempool.
// This function is safe for concurrent access.
func (m *MemPool) GetAllTxs() []*transactions.Stealth {
	m.lock.RLock()
	txs := make([]*transactions.Stealth, len(m.txs))
	i := 0
	for _, tx := range m.txs {
		txs[i] = tx
		i++
	}

	m.lock.RUnlock()
	return txs
}

// Clear the mempool.
// This function is safe for concurrent access.
func (m *MemPool) Clear() {
	m.lock.Lock()
	m.txs = make(map[string]*transactions.Stealth)
	m.lock.Unlock()
}

func (m *MemPool) addTx(tx *transactions.Stealth) {
	hex := hex.EncodeToString(tx.Hash)
	m.lock.Lock()
	m.txs[hex] = tx
	m.lock.Unlock()
}

func (m *MemPool) removeTx(tx *transactions.Stealth) {
	hex := hex.EncodeToString(tx.Hash)
	m.lock.Lock()
	m.txs[hex] = nil
	m.lock.Unlock()
}
