// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"sort"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

type (
	keyFee struct {
		k txHash
		f uint64
	}

	// HashMap represents a pool implementation based on golang map. The generic
	// solution to bench against.
	HashMap struct {
		// transactions pool.
		lock *sync.RWMutex
		data map[txHash]TxDesc

		// sorted is data keys sorted by Fee in a descending order
		// sorting happens at point of accepting new entry in order to allow
		// Block Generator to fetch highest-fee txs without delays in sorting.
		sorted []keyFee

		// spent key images from the transactions in the pool
		// spentkeyImages map[keyImage]bool.
		Capacity uint32
		txsSize  uint32
	}
)

func (m *HashMap) Create(path string) error {
	m.data = make(map[txHash]TxDesc, m.Capacity)
	m.sorted = make([]keyFee, 0, m.Capacity)

	return nil
}

// Put sets the value for the given key. It overwrites any previous value
// for that key.
func (m *HashMap) Put(t TxDesc) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// store tx
	txID, err := t.tx.CalculateHash()
	if err != nil {
		return err
	}

	var k txHash
	copy(k[:], txID)

	// Let's check if we're not inserting a double key. This is, is essence,
	// fine for the map, but having a duplicate entry in `sorted` can cause
	// a panic in CalculateRoot, in the block generator.

	_, ok := m.data[k]
	if ok {
		return ErrAlreadyExists
	}

	m.data[k] = t
	m.txsSize += uint32(t.size)

	// sort keys by Fee
	// Bulk sort like (sort.Slice) performs a few times slower than
	// a simple binarysearch&shift algorithm.
	_, fee := t.tx.Values()

	index := sort.Search(len(m.sorted), func(i int) bool {
		return m.sorted[i].f < fee
	})

	m.sorted = append(m.sorted, keyFee{})

	copy(m.sorted[index+1:], m.sorted[index:])

	m.sorted[index] = keyFee{k: k, f: fee}
	return nil
}

// Clone the entire pool.
func (m HashMap) Clone() []transactions.ContractCall {
	m.lock.RLock()
	defer m.lock.RUnlock()

	r := make([]transactions.ContractCall, len(m.data))
	i := 0

	for _, t := range m.data {
		r[i] = t.tx
		i++
	}

	return r
}

// FilterByType returns all transactions for a specific type that are
// currently in the HashMap.
func (m HashMap) FilterByType(filterType transactions.TxType) []transactions.ContractCall {
	m.lock.RLock()
	defer m.lock.RUnlock()

	txs := make([]transactions.ContractCall, 0)

	for _, t := range m.data {
		if t.tx.Type() == filterType {
			txs = append(txs, t.tx)
		}
	}

	return txs
}

// Contains returns true if the given key is in the pool.
func (m *HashMap) Contains(txID []byte) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var k txHash

	copy(k[:], txID)
	_, ok := m.data[k]

	return ok
}

// Get returns a tx for a given txID if it exists.
func (m *HashMap) Get(txID []byte) transactions.ContractCall {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var k txHash

	copy(k[:], txID)

	txd, ok := m.data[k]
	if !ok {
		return nil
	}

	return txd.tx
}

// Delete a key in the hashmap.
func (m *HashMap) Delete(txID []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()

	var k txHash

	copy(k[:], txID)

	tx, ok := m.data[k]
	if !ok {
		return
	}

	m.txsSize -= uint32(tx.size)

	delete(m.data, k)

	// TODO: this is naive, and may be improved upon.
	for i, entry := range m.sorted {
		if entry.k == k {
			m.sorted = append(m.sorted[:i], m.sorted[i+1:]...)
		}
	}
}

// Size of the txs.
func (m *HashMap) Size() uint32 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.txsSize
}

// Len returns the number of tx entries.
func (m *HashMap) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.data)
}

// Range iterates through all tx entries.
func (m *HashMap) Range(fn func(k txHash, t TxDesc) error) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for k, v := range m.data {
		err := fn(k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

// RangeSort iterates through all tx entries sorted by Fee
// in a descending order.
func (m *HashMap) RangeSort(fn func(k txHash, t TxDesc) (bool, error)) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, value := range m.sorted {
		done, err := fn(value.k, m.data[value.k])
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}

	return nil
}
