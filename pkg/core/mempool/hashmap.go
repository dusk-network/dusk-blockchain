package mempool

import (
	"fmt"
	"sort"

	"github.com/dusk-network/dusk-wallet/transactions"
)

const (
	keyImageSize = 32
)

type (
	keyImage [keyImageSize]byte

	keyFee struct {
		k txHash
		f uint64
	}

	// HashMap represents a pool implementation based on golang map. The generic
	// solution to bench against.
	HashMap struct {
		// transactions pool
		data map[txHash]TxDesc

		// sorted is data keys sorted by Fee in a descending order
		// sorting happens at point of accepting new entry in order to allow
		// Block Generator to fetch highest-fee txs without delays in sorting
		sorted []keyFee

		// spent key images from the transactions in the pool
		spentkeyImages map[keyImage]bool
		Capacity       uint32
		txsSize        uint32
	}
)

// Put sets the value for the given key. It overwrites any previous value
// for that key;
func (m *HashMap) Put(t TxDesc) error {

	if m.data == nil {
		m.data = make(map[txHash]TxDesc, m.Capacity)
		m.sorted = make([]keyFee, 0, m.Capacity)
	}

	if m.spentkeyImages == nil {
		m.spentkeyImages = make(map[keyImage]bool)
	}

	// store tx
	txID, err := t.tx.CalculateHash()
	if err != nil {
		return err
	}

	var k txHash
	copy(k[:], txID)
	m.data[k] = t

	m.txsSize += uint32(t.size)

	// sort keys by Fee
	// Bulk sort like (sort.Slice) performs a few times slower than
	// a simple binarysearch&shift algorithm.
	fee := t.tx.StandardTx().Fee.BigInt().Uint64()

	index := sort.Search(len(m.sorted), func(i int) bool {
		return m.sorted[i].f < fee
	})

	m.sorted = append(m.sorted, keyFee{})
	copy(m.sorted[index+1:], m.sorted[index:])
	m.sorted[index] = keyFee{k: k, f: fee}

	// store all tx key images, if provided
	for i, input := range t.tx.StandardTx().Inputs {
		if len(input.KeyImage.Bytes()) == keyImageSize {
			var ki keyImage
			copy(ki[:], input.KeyImage.Bytes())
			m.spentkeyImages[ki] = true
		} else {
			return fmt.Errorf("invalid key image found at index %d", i)
		}
	}

	return nil
}

// Clone the entire pool
func (m HashMap) Clone() []transactions.Transaction {

	r := make([]transactions.Transaction, len(m.data))
	i := 0
	for _, t := range m.data {
		r[i] = t.tx
		i++
	}

	return r
}

// Contains returns true if the given key is in the pool.
func (m *HashMap) Contains(txID []byte) bool {
	var k txHash
	copy(k[:], txID)
	_, ok := m.data[k]
	return ok
}

// Size of the txs
func (m *HashMap) Size() uint32 {
	return m.txsSize
}

// Len returns the number of tx entries
func (m *HashMap) Len() int {
	return len(m.data)
}

// Range iterates through all tx entries
func (m *HashMap) Range(fn func(k txHash, t TxDesc) error) error {

	for k, v := range m.data {
		err := fn(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// RangeSort iterates through all tx entries sorted by Fee
// in a descending order
func (m *HashMap) RangeSort(fn func(k txHash, t TxDesc) (bool, error)) error {

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

// ContainsKeyImage returns true if txpool includes a input that contains
// this keyImage
func (m *HashMap) ContainsKeyImage(txInputKeyImage []byte) bool {
	var ki keyImage
	copy(ki[:], txInputKeyImage)
	_, ok := m.spentkeyImages[ki]
	return ok
}
