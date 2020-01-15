package mempool

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/transactions"
)

func TestSortedKeys(t *testing.T) {

	pool := &HashMap{Capacity: 100}

	// Generate 100 random txs
	for i := 0; i < 100; i++ {

		tx := helper.RandomStandardTx(t, false)

		randFee := big.NewInt(0).SetUint64(uint64(rand.Intn(10000)))
		tx.Fee.SetBigInt(randFee)

		td := TxDesc{tx: tx}
		if err := pool.Put(td); err != nil {
			t.Fatal(err.Error())
		}
	}

	// Iterate through all tx expecting each one has lower fee than
	// the previos one
	var prevVal uint64
	prevVal = math.MaxUint64

	err := pool.RangeSort(func(k txHash, t TxDesc) (bool, error) {

		val := t.tx.StandardTx().Fee.BigInt().Uint64()
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

		tx := helper.RandomStandardTx(t, false)

		constFee := big.NewInt(0).SetUint64(20)
		tx.Fee.SetBigInt(constFee)

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
	pool := HashMap{Capacity: 100}

	// Generate 100 random txs
	hashes := make([][]byte, 100)
	for i := 0; i < 100; i++ {

		tx := helper.RandomStandardTx(t, false)
		tx.CalculateHash()
		hashes[i] = tx.TxID

		constFee := big.NewInt(0).SetUint64(20)
		tx.Fee.SetBigInt(constFee)

		td := TxDesc{tx: tx, received: time.Now()}
		if err := pool.Put(td); err != nil {
			t.Fatal(err.Error())
		}
	}

	// Get a random tx from the pool
	n := rand.Intn(100)
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
			if !pool.Contains(txs[i].TxID) {
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

func dummyTransactionsSet(size int) []*transactions.Standard {

	txs := make([]*transactions.Standard, size)
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
}
