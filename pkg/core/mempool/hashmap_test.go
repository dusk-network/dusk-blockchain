package mempool

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
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

	err := pool.RangeSort(func(k txHash, t TxDesc) error {

		val := t.tx.StandardTx().Fee.BigInt().Uint64()
		if prevVal < val {
			return errors.New("keys not in a descending order")
		}

		prevVal = val
		return nil
	})

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestStableSortedKeys(t *testing.T) {

	pool := &HashMap{Capacity: 100}

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
	err := pool.RangeSort(func(k txHash, t TxDesc) error {

		val := t.received
		if prevReceived.After(val) {
			return errors.New("order of receiving should be kept")
		}

		prevReceived = val
		return nil
	})

	if err != nil {
		t.Fatalf(err.Error())
	}
}
