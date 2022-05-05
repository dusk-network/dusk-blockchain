// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"bytes"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/tidwall/buntdb"
)

const (
	feePrefix       = "fi:"
	feeIndex        = "fee_index"
	itemsCountPerTx = 2
)

const (
	needFullTx uint = iota
	needTxSizeOnly
)

type (
	// buntdbPool implements Pool interface to provide an in-memory transactions storage
	// with EverySecond sync policy.
	buntdbPool struct {
		db *buntdb.DB

		// Cumulative size of all added transactions
		cumulativeTxsSize uint32
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

	// Fast and safer sync policy.
	// In addition, syncing is done on closing mempool.
	config.SyncPolicy = buntdb.EverySecond

	// Auto-shrink should be always enabled
	config.AutoShrinkDisabled = false

	if err := db.SetConfig(config); err != nil {
		_ = db.Close()
		return err
	}

	m.db = db

	if err := m.createIndices(); err != nil {
		log.WithError(err).Warn("could not create indices")
	}

	// Sum cumulative trasactions size
	cumulativeTxsSize := uint32(0)
	_ = db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			buf := bytes.NewBufferString(value)

			txd, err := unmarshalTxDesc(buf, needTxSizeOnly)
			if err != nil {
				panic(err)
			}

			cumulativeTxsSize += uint32(txd.size)
			return true // continue iteration
		})
		return err
	})

	atomic.StoreUint32(&m.cumulativeTxsSize, cumulativeTxsSize)
	return nil
}

// Put adds new transaction to the pool. It creates two key-value pairs per a tx.
// transacton_id -> transaction data (marshaled)
// fi:transacton_id -> transaction fee value
func (m *buntdbPool) Put(t TxDesc) error {
	var key, value bytes.Buffer
	var err error

	txID, err := t.tx.CalculateHash()
	if err != nil {
		return err
	}

	key.Write(txID)

	if e := marshalTxDesc(&value, &t); e != nil {
		return e
	}

	err = m.db.Update(func(tx *buntdb.Tx) error {
		if _, _, err = tx.Set(key.String(), value.String(), nil); err != nil {
			return err
		}

		feeKey := feePrefix + key.String()

		var fee uint64

		fee, err = t.tx.Fee()
		if err != nil {
			log.WithError(err).Warn("fee could not be read")
		}

		_, _, err = tx.Set(feeKey, strconv.FormatInt(int64(fee), 10), nil)
		if err != nil {
			return err
		}

		atomic.AddUint32(&m.cumulativeTxsSize, uint32(t.size))
		return nil
	})

	return err
}

// Contain returns true if the given key is in the pool.
func (m *buntdbPool) Contain(txID []byte) bool {
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
	t, _ := m.getTxDesc(txID, needFullTx)
	return t.tx
}

// getTxDesc returns a tx for a given txID if it exists.
func (m *buntdbPool) getTxDesc(txID []byte, need uint) (TxDesc, error) {
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
		return TxDesc{}, err
	}

	buf := bytes.NewBufferString(value)

	txdesc, err = unmarshalTxDesc(buf, need)
	if err != nil {
		return TxDesc{}, err
	}

	return txdesc, nil
}

// Delete a transaction by id.
func (m *buntdbPool) Delete(txID []byte) error {
	var (
		key bytes.Buffer
		err error
	)

	key.Write(txID)

	// For the purpose of deleting a transaction we need to recalculate cumulativeTxsSize.
	// getTxDesc is called with needTxSizeOnly to ensure we read tx size only but not entire tx.
	desc, err := m.getTxDesc(txID, needTxSizeOnly)
	if err != nil {
		return errNotFound
	}

	size := int(desc.size)

	err = m.db.Update(func(t *buntdb.Tx) error {
		_, err = t.Delete(key.String())
		if err != nil {
			return err
		}

		_, err = t.Delete(feePrefix + key.String())
		if err != nil {
			return err
		}

		// subtract deleted tx size
		atomic.AddUint32(&m.cumulativeTxsSize, ^uint32(size-1))
		return err
	})

	return err
}

// Range iterates through all tx entries ordered by transaction ids.
func (m *buntdbPool) Range(fn func(k txHash, t TxDesc) error) error {
	err := m.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			buf := bytes.NewBufferString(value)

			txdesc, err := unmarshalTxDesc(buf, needFullTx)
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
	return atomic.LoadUint32(&m.cumulativeTxsSize)
}

// Len returns the number of tx entries.
func (m *buntdbPool) Len() int {
	var len int

	_ = m.db.View(func(tx *buntdb.Tx) error {
		len, _ = tx.Len()
		return nil
	})

	return len / itemsCountPerTx
}

// RangeSort iterates through all tx entries sorted by Fee
// in a descending order.
func (m *buntdbPool) RangeSort(fn func(k txHash, t TxDesc) (bool, error)) error {
	return m.db.View(func(tx *buntdb.Tx) error {
		// Iterate keys sorted by fee.
		// For each key, get marshaled tx data
		err := tx.Descend(feeIndex, func(feeKey, fee string) bool {
			txid := feeKey[len(feePrefix):]

			// Get full transaction data
			value, err := tx.Get(txid)
			if err != nil {
				return false
			}

			// unmarshal
			buf := bytes.NewBufferString(value)

			txdesc, err := unmarshalTxDesc(buf, needFullTx)
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

// ContainNullifier implements Pool.ContainAnyNullifiers.
func (m *buntdbPool) ContainAnyNullifiers(nullifiers [][]byte) (bool, []byte) {
	var repeatedNullifier []byte

	_ = m.db.View(func(tx *buntdb.Tx) error {
		_ = tx.Ascend("", func(key, value string) bool {
			t, err := unmarshalTxDesc(bytes.NewBufferString(value), needFullTx)
			if err != nil {
				return true
			}

			d, err := t.tx.Decode()
			if err != nil {
				return true
			}

			for _, n := range d.Nullifiers {
				for _, t := range nullifiers {
					if bytes.Equal(n, t) {
						repeatedNullifier = t
						// we found it, discontinue iterating
						return false
					}
				}
			}

			return true
		})
		return nil
	})

	return len(repeatedNullifier) != 0, repeatedNullifier
}

// Clone the entire pool.
func (m *buntdbPool) Clone() []transactions.ContractCall {
	// Not in use
	return nil
}

// FilterByType returns all transactions for a specific type that are
// currently in the HashMap.
func (m *buntdbPool) FilterByType(filterType transactions.TxType) []transactions.ContractCall {
	// Not in use.
	return nil
}

// GetTxsByNullifier implements Pool.GetTxsByNullifier. Later on, the execution time could
// be improved by adding nullifier index, if needed.
func (m *buntdbPool) GetTxsByNullifier(nullifier []byte) ([][]byte, error) {
	found := make([][]byte, 0)

	err := m.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			buf := bytes.NewBufferString(value)

			txdesc, err := unmarshalTxDesc(buf, needFullTx)
			if err != nil {
				panic(err)
			}

			d, err := txdesc.tx.Decode()
			if err != nil {
				// continue iterating
				return true
			}

			for _, n := range d.Nullifiers {
				if bytes.Equal(n, nullifier) {
					hash := bytes.NewBufferString(key).Bytes()
					found = append(found, hash)
					break
				}
			}

			return true // continue iteration
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	if len(found) == 0 {
		return nil, errors.New("not found")
	}

	return found, nil
}

func (m *buntdbPool) createIndices() error {
	var found bool
	// Create index for sorting txids by fee
	indexList, err := m.db.Indexes()
	if err != nil {
		return err
	}

	for _, name := range indexList {
		if name != feeIndex {
			continue
		}

		found = true
		break
	}

	if !found {
		pattern := feePrefix + "*"
		// An error will occur if an index with the same name already exists.
		if err := m.db.CreateIndex(feeIndex, pattern, buntdb.IndexInt); err != nil {
			return err
		}
	}

	return nil
}

func (m buntdbPool) Close() {
	if err := m.db.Close(); err != nil {
		log.WithError(err).Warn("buntdb close with error")
	}
}

func marshalTxDesc(r *bytes.Buffer, p *TxDesc) error {
	if err := encoding.WriteUint32LE(r, uint32(p.size)); err != nil {
		return err
	}

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

func unmarshalTxDesc(r *bytes.Buffer, need uint) (TxDesc, error) {
	var size uint32
	if err := encoding.ReadUint32LE(r, &size); err != nil {
		return TxDesc{}, err
	}

	if need == needTxSizeOnly {
		return TxDesc{
			size: uint(size),
		}, nil
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
