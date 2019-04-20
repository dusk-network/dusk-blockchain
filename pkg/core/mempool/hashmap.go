package mempool

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"unsafe"
)

type key [32]byte

// Pool implementation based on golang map. The generic solution to bench
// against.
type HashMap struct {
	data     map[key]TxDesc
	Capacity uint32
	txsSize  uint64
}

// Put sets the value for the given key. It overwrites any previous value
// for that key;
func (m *HashMap) Put(t TxDesc) error {

	if m.data == nil {
		m.data = make(map[key]TxDesc, m.Capacity)
	}

	txID, _ := t.tx.CalculateHash()
	var k key
	copy(k[:], txID)
	m.data[k] = t

	m.txsSize += uint64(unsafe.Sizeof(t.tx))

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
func (m *HashMap) Size() uint32 {
	return uint32(m.txsSize / 1e6)
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
