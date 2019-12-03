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

		// data keys sorted by Fee in a descending order
		sorted []keyFee

		// spent key images from the transactions in the pool
		spentkeyImages map[keyImage]bool
		Capacity       uint32
		txsSize        uint64
	}
)

// Put sets the value for the given key. It overwrites any previous value
// for that key;
func (m *HashMap) Put(t TxDesc) error {

	if m.data == nil {
		m.data = make(map[txHash]TxDesc, m.Capacity)
	}

	if m.spentkeyImages == nil {
		// TODO: consider capacity value here
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

	m.txsSize += uint64(t.size)

	// sort keys by Fee
	fee := t.tx.StandardTx().Fee.BigInt().Uint64()
	m.sorted = append(m.sorted, keyFee{k: k, f: fee})

	sort.SliceStable(m.sorted, func(i, j int) bool {
		return m.sorted[i].f > m.sorted[j].f
	})

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
func (m *HashMap) Size() float64 {
	return float64(m.txsSize) / (1024 * 1024)
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
func (m *HashMap) RangeSort(fn func(k txHash, t TxDesc) error) error {

	for _, value := range m.sorted {
		err := fn(value.k, m.data[value.k])
		if err != nil {
			return err
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
