// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"bytes"
	"encoding/hex"
	"math"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	assert "github.com/stretchr/testify/require"
)

func createTemp(pattern string) string {
	f, _ := os.CreateTemp(os.TempDir(), pattern)
	dbpath := f.Name()
	f.Close()

	return dbpath
}

func TestBuntStoreGet(t *testing.T) {
	assert := assert.New(t)

	dbpath := createTemp("file1.db")
	defer os.Remove(dbpath)

	pool := buntdbPool{}
	pool.Create(dbpath)

	// Generate 10 random txs
	mockTxs := make([]TxDesc, 0)

	for i := 0; i < 10; i++ {
		tx := transactions.RandTx()
		td := TxDesc{tx: tx, received: time.Now()}

		assert.NoError(pool.Put(td))
		mockTxs = append(mockTxs, td)
	}

	assert.True(pool.Len() == 10)

	// Get a random tx from the pool
	for _, td := range mockTxs {
		hash, err := td.tx.CalculateHash()
		if err != nil {
			t.Fail()
		}

		cc := pool.Get(hash)

		hash2, _ := cc.CalculateHash()
		assert.True(bytes.Equal(hash, hash2))

		// Check Delete / Contains
		assert.True(pool.Contains(hash))

		pool.Delete(hash)
		assert.False(pool.Contains(hash))
	}

	// Now get a tx for a hash that is not in the pool
	hash, _ := crypto.RandEntropy(32)
	assert.Nil(pool.Get(hash))
}

func TestBuntSortedKeys(tst *testing.T) {
	assert := assert.New(tst)

	dbpath := createTemp("file2.db")
	defer os.Remove(dbpath)

	pool := buntdbPool{}
	pool.Create(dbpath)

	// Generate 10 random txs
	for i := 0; i < 10; i++ {
		tx := transactions.RandTx()
		td := TxDesc{tx: tx}
		assert.NoError(pool.Put(td))
	}

	// Iterate through all tx expecting each one has lower fee than
	// the previous one
	var prevVal uint64
	prevVal = math.MaxUint64

	pool.RangeSort(func(k txHash, t TxDesc) (bool, error) {
		fee := t.tx.Fee()

		tst.Log(hex.EncodeToString(k[:]), "_", fee)

		if prevVal < fee {
			assert.Fail("keys not in a descending order")
		}

		prevVal = fee
		return false, nil
	})

	assert.NotEqual(prevVal != math.MaxUint64, "rangesort not called")
}
