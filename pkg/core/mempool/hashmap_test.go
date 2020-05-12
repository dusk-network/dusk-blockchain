package mempool

import (
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	assert "github.com/stretchr/testify/require"
)

func TestSortedKeys(t *testing.T) {
	assert := assert.New(t)

	pool := &HashMap{Capacity: 100}

	// Generate 100 random txs
	for i := 0; i < 100; i++ {
		tx := transactions.RandTx()
		td := TxDesc{tx: tx}
		assert.NoError(pool.Put(td))
	}

	// Iterate through all tx expecting each one has lower fee than
	// the previous one
	var prevVal uint64
	prevVal = math.MaxUint64

	err := pool.RangeSort(func(k txHash, t TxDesc) (bool, error) {

		val := t.tx.StandardTx().Fee.Value
		if prevVal < val {
			return false, errors.New("keys not in a descending order")
		}

		prevVal = val
		return false, nil
	})

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestStableSortedKeys(t *testing.T) {

	pool := HashMap{Capacity: 100}

	// Generate 100 random txs
	for i := 0; i < 100; i++ {

		amount := transactions.RandUint64()
		bf := transactions.RandBlind()
		tx := transactions.MockTx(amount, uint64(20), false, bf)
		td := TxDesc{tx: tx, received: time.Now()}
		if err := pool.Put(td); err != nil {
			t.Fatal(err.Error())
		}
	}

	// Iterate through all tx expecting order of receiving is kept when
	// tx has same fee
	var prevReceived time.Time
	err := pool.RangeSort(func(k txHash, t TxDesc) (bool, error) {

		val := t.received
		if prevReceived.After(val) {
			return false, errors.New("order of receiving should be kept")
		}

		prevReceived = val
		return false, nil
	})

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestGet(t *testing.T) {
	txsCount := 10
	pool := HashMap{Capacity: uint32(txsCount)}

	// Generate 10 random txs
	hashes := make([][]byte, txsCount)
	for i := 0; i < txsCount; i++ {

		amount := transactions.RandUint64()
		bf := transactions.RandBlind()
		tx := transactions.MockTx(amount, uint64(20), false, bf)
		hash, _ := tx.CalculateHash()
		hashes[i] = hash

		td := TxDesc{tx: tx, received: time.Now()}
		if err := pool.Put(td); err != nil {
			t.Fatal(err.Error())
		}
	}

	// Get a random tx from the pool
	n := rand.Intn(txsCount)
	tx := pool.Get(hashes[n])
	if tx == nil {
		t.Fatal("tx is not supposed to be nil")
	}

	// Now get a tx for a hash that is not in the pool
	hash, _ := crypto.RandEntropy(32)
	tx = pool.Get(hash)
	if tx != nil {
		t.Fatal("should not have gotten a tx")
	}
}

func BenchmarkPut(b *testing.B) {

	txs := dummyTransactionsSet(50000)
	b.ResetTimer()

	// Put all transactions
	for tN := 0; tN < b.N; tN++ {
		pool := HashMap{Capacity: uint32(len(txs))}
		for i := 0; i < len(txs); i++ {

			td := TxDesc{tx: txs[i], received: time.Now(), size: uint(i)}
			if err := pool.Put(td); err != nil {
				b.Fatalf(err.Error())
			}
		}

		b.Logf("Pool number of txs: %d", pool.Len())
	}
}

func BenchmarkContains(b *testing.B) {

	txs := dummyTransactionsSet(50000)

	// Put all transactions
	pool := HashMap{Capacity: uint32(len(txs))}
	for i := 0; i < len(txs); i++ {

		td := TxDesc{tx: txs[i], received: time.Now(), size: uint(i)}
		if err := pool.Put(td); err != nil {
			b.Fatalf(err.Error())
		}
	}

	b.ResetTimer()

	for tN := 0; tN < b.N; tN++ {
		for i := 0; i < len(txs); i++ {
			txid, _ := txs[i].CalculateHash()
			if !pool.Contains(txid) {
				b.Fatal("missing tx")
			}
		}
	}

	b.Logf("Pool number of txs: %d", pool.Len())
}

func BenchmarkRangeSort(b *testing.B) {

	txs := dummyTransactionsSet(10000)

	// Put all transactions
	pool := HashMap{Capacity: uint32(len(txs))}
	for i := 0; i < len(txs); i++ {

		td := TxDesc{tx: txs[i], received: time.Now(), size: uint(i)}
		if err := pool.Put(td); err != nil {
			b.Fatalf(err.Error())
		}
	}

	b.ResetTimer()

	for tN := 0; tN < b.N; tN++ {
		err := pool.RangeSort(func(k txHash, t TxDesc) (bool, error) {
			return false, nil
		})

		if err != nil {
			b.Fatalf(err.Error())
		}
	}

	b.Logf("Pool number of txs: %d", pool.Len())
}

func dummyTransactionsSet(size int) []transactions.ContractCall {
	/*
		txs := make([]transactions.ContractCall, size)
		// Generate N random tx
		dummyTx, _ := transactions.NewStandard(0, 2, 0)
		for i := 0; i < len(txs); i++ {

			// change fee to enable sorting
			randFee := big.NewInt(0).SetUint64(uint64(rand.Intn(1000000)))
			dummyTx.Fee.SetBigInt(randFee)

			clone := *dummyTx
			clone.TxID, _ = crypto.RandEntropy(32)

			txs[i] = &clone
		}

		return txs
	*/
	return nil
}
