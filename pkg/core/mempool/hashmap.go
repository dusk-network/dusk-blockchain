package mempool

import (
	"fmt"
	"unsafe"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

const (
	keyImageSize = 32
)

type (
	key      [32]byte
	keyImage [keyImageSize]byte

	// HashMap represents a pool implementation based on golang map. The generic
	// solution to bench against.
	HashMap struct {
		// transactions pool
		data map[key]TxDesc
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
		m.data = make(map[key]TxDesc, m.Capacity)
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

	var k key
	copy(k[:], txID)
	m.data[k] = t

	m.txsSize += uint64(unsafe.Sizeof(t.tx))

	// store all tx key images, if provided
	for i, input := range t.tx.StandardTX().Inputs {
		if len(input.KeyImage) == keyImageSize {
			var ki keyImage
			copy(ki[:], input.KeyImage)
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
	var k key
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
func (m *HashMap) Range(fn func(k key, t TxDesc) error) error {
	for k, v := range m.data {
		err := fn(k, v)
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
