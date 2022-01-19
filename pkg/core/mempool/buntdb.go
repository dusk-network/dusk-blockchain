// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/tidwall/buntdb"
)

type (
	// buntdbPool implements Pool interface to provide an in-memory transactions storage
	// with EverySecond sync policy.
	buntdbPool struct {
		db *buntdb.DB
	}
)

// newBuntdbPool opens/creates buntdb file with buntdb.EverySecond config.
// fee_index is created to sort transaction ids by fee.
func (m *buntdbPool) Create(path string) error {
	db, errOpen := buntdb.Open(path)
	if errOpen != nil {
		return errOpen
	}

	var config buntdb.Config
	if err := db.ReadConfig(&config); err != nil {
		_ = db.Close()
		return err
	}

	config.SyncPolicy = buntdb.EverySecond

	if err := db.SetConfig(config); err != nil {
		_ = db.Close()
		return err
	}

	// Create index for sorting txids by fee
	if err := db.CreateIndex("fee_index", "fi:*", buntdb.IndexInt); err != nil {
		return err
	}

	m.db = db
	return nil
}

// Put adds new transaction to the pool. It creates two key-value pairs per a tx.
// transacton_id -> transaction data (marshaled)
// fi:transacton_id -> transaction fee value
func (m *buntdbPool) Put(t TxDesc) error {
	var key, value bytes.Buffer

	txID, err := t.tx.CalculateHash()
	if err != nil {
		return err
	}

	key.Write(txID)

	if e := marshalTxDesc(&value, &t); e != nil {
		return e
	}

	err = m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(key.String(), value.String(), nil)
		if err != nil {
			return err
		}

		feeKey := fmt.Sprintf("fi:%s", key.String())
		_, fee := t.tx.Values()

		_, _, err = tx.Set(feeKey, strconv.FormatInt(int64(fee), 10), nil)
		if err != nil {
			return err
		}

		return err
	})

	return err
}

// Contains returns true if the given key is in the pool.
func (m *buntdbPool) Contains(txID []byte) bool {
	var (
		key bytes.Buffer
		err error
	)

	key.Write(txID)

	err = m.db.View(func(t *buntdb.Tx) error {
		_, err = t.Get(key.String())
		if err != nil {
			return err
		}

		return err
	})

	return err == nil
}

// Get returns a tx for a given txID if it exists.
func (m *buntdbPool) Get(txID []byte) transactions.ContractCall {
	var (
		key    bytes.Buffer
		value  string
		err    error
		txdesc TxDesc
	)

	key.Write(txID)

	err = m.db.View(func(t *buntdb.Tx) error {
		value, err = t.Get(key.String())
		if err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return nil
	}

	buf := bytes.NewBufferString(value)

	txdesc, err = unmarshalTxDesc(buf)
	if err != nil {
		return nil
	}

	return txdesc.tx
}

// Delete a transaction by id.
func (m *buntdbPool) Delete(txID []byte) {
	var (
		key bytes.Buffer
		err error
	)

	key.Write(txID)

	err = m.db.Update(func(t *buntdb.Tx) error {
		_, _ = t.Delete(key.String())
		if err != nil {
			return err
		}

		return err
	})
}

// Range iterates through all tx entries ordered by transaction ids.
func (m *buntdbPool) Range(fn func(k txHash, t TxDesc) error) error {
	err := m.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			buf := bytes.NewBufferString(value)

			txdesc, err := unmarshalTxDesc(buf)
			if err != nil {
				panic(err)
			}

			var t txHash
			copy(t[:], bytes.NewBufferString(key).Bytes())

			if err := fn(t, txdesc); err != nil {
				// discontinue iteration
				return false
			}

			return true // continue iteration
		})
		return err
	})

	return err
}

// Size of the txs.
func (m *buntdbPool) Size() uint32 {
	// TODO:
	return 0
}

// Len returns the number of tx entries.
func (m *buntdbPool) Len() int {
	// TODO:
	return 0
}

// RangeSort iterates through all tx entries sorted by Fee
// in a descending order.
func (m *buntdbPool) RangeSort(fn func(k txHash, t TxDesc) (bool, error)) error {
	return m.db.View(func(tx *buntdb.Tx) error {
		// Iterate keys sorted by fee.
		// For each key, get marshaled tx
		err := tx.Descend("fee_index", func(feeKey, fee string) bool {
			txid := feeKey[3:]

			// Get full transaction data
			value, err := tx.Get(txid)
			if err != nil {
				return false
			}

			// unmarshal
			buf := bytes.NewBufferString(value)

			txdesc, err := unmarshalTxDesc(buf)
			if err != nil {
				panic(err)
			}

			var t txHash
			copy(t[:], bytes.NewBufferString(txid).Bytes())

			done, err := fn(t, txdesc)
			if err != nil || done {
				// discontinue iteration
				return false
			}

			return true // continue iteration
		})
		return err
	})
}

// Clone the entire pool.
func (m buntdbPool) Clone() []transactions.ContractCall {
	// Not in use
	return nil
}

// FilterByType returns all transactions for a specific type that are
// currently in the HashMap.
func (m buntdbPool) FilterByType(filterType transactions.TxType) []transactions.ContractCall {
	// Not in use.
	return nil
}

func marshalTxDesc(r *bytes.Buffer, p *TxDesc) error {
	if err := encoding.WriteUint32LE(r, uint32(p.size)); err != nil {
		return err
	}

	// TODO: new file

	if err := encoding.WriteUint64LE(r, uint64(p.received.Unix())); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, uint64(p.verified.Unix())); err != nil {
		return err
	}

	if err := transactions.Marshal(r, p.tx); err != nil {
		return err
	}

	return nil
}

func unmarshalTxDesc(r *bytes.Buffer) (TxDesc, error) {
	var size uint32
	if err := encoding.ReadUint32LE(r, &size); err != nil {
		return TxDesc{}, err
	}

	var received uint64
	if err := encoding.ReadUint64LE(r, &received); err != nil {
		return TxDesc{}, err
	}

	var verified uint64
	if err := encoding.ReadUint64LE(r, &verified); err != nil {
		return TxDesc{}, err
	}

	c := transactions.NewTransaction()
	if err := transactions.Unmarshal(r, c); err != nil {
		return TxDesc{}, err
	}

	return TxDesc{
		tx:       c,
		size:     uint(size),
		verified: time.Unix(int64(verified), 0),
		received: time.Unix(int64(received), 0),
	}, nil
}
