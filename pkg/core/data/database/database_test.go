package database

import (
	"bytes"
	"os"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	assert "github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

const path = "mainnet"

//TODO: #446 , shall this be refactored ?
func TestPutGet(t *testing.T) {
	assert := assert.New(t)

	// New
	db, err := New(path)
	assert.NoError(err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.NoError(err)

	// Close and re-open database
	err = db.Close()
	assert.NoError(err)
	db, err = New(path)
	assert.NoError(err)

	// Get
	val, err := db.Get(key)
	assert.NoError(err)
	assert.True(bytes.Equal(val, value))

	// Delete
	err = db.Delete(key)
	assert.NoError(err)

	// Get after delete
	val, err = db.Get(key)
	assert.Equal(leveldb.ErrNotFound, err)
	assert.True(bytes.Equal(val, []byte{}))
}

func TestPutFetchTxRecord(t *testing.T) {
	// TODO: rework for RUSK integration
	/*
		// New
		db, e := New(path)
		if !assert.NoError(t, e) {
			t.FailNow()
		}

		// Make sure to delete this dir after test
		defer os.RemoveAll(path)

		// Create some random txs
		txs := make([]transactions.ContractCall, 10)
		privViews := make([]*key.PrivateView, 10)
		for i := range txs {
			tx, privView := randTxForRecord(transactions.TxType(i % 5))
			privViews[i] = privView
			txs[i] = tx
			if err := db.PutTxRecord(tx, 20, txrecords.Direction(i%2), privView); err != nil {
				t.Fatal(err)
			}
		}

		// Fetch records
		records, err := db.FetchTxRecords()
		if err != nil {
			t.Fatal(err)
		}

		// Check correctness
		assert.Equal(t, len(txs), len(records))
		checked := 0
		for _, record := range records {
			// Find out which tx this is
			for i, tx := range txs {
				if hex.EncodeToString(tx.StandardTx().Outputs[0]) == record.Recipient {
					assert.Equal(t, tx.LockTime(), record.UnlockHeight-record.Height)
					assert.Equal(t, tx.Type(), record.TxType)
					amount := tx.StandardTx().Outputs[0].EncryptedAmount
					if transactions.ShouldEncryptValues(tx) {
						amount = transactions.DecryptAmount(amount, tx.StandardTx().R, 0, *privViews[i])
					}

					assert.Equal(t, amount.BigInt().Uint64(), record.Amount)
					checked++
				}
			}
		}

		assert.Equal(t, len(txs), checked)
	*/
}

func TestClear(t *testing.T) {
	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.NoError(t, err)

	// Empty out database
	assert.NoError(t, db.Clear())

	// Info should now be gone entirely
	_, err = db.Get([]byte("hello"))
	assert.Error(t, err)
}

func randTxForRecord(t transactions.TxType) transactions.ContractCall { //nolint
	// TODO: rework for RUSK integration
	/*
		var tx transactions.Transaction
		switch t {
		case transactions.StandardType:
			tx, _ = transactions.NewStandard(0, 1, 100)
		case transactions.TimelockType:
			tx, _ = transactions.NewTimelock(0, 1, 100, 10000)
		case transactions.BidType:
			tx, _ = transactions.NewBid(0, 1, 100, 5000, make([]byte, 32))
		case transactions.StakeType:
			tx, _ = transactions.NewStake(0, 1, 100, 2130, make([]byte, 129))
		case transactions.CoinbaseType:
			tx = transactions.NewCoinbase(make([]byte, 100), make([]byte, 32), 2)
		}

		var amount ristretto.Scalar
		amount.Rand()
		seed := make([]byte, 64)
		rand.Read(seed)
		keyPair := key.NewKeyPair(seed)
		privView, err := keyPair.PrivateView()
		if err != nil {
			panic(err)
		}

		addr, err := keyPair.PublicKey().PublicAddress(1)
		if err != nil {
			panic(err)
		}

		if tx.Type() == transactions.CoinbaseType {
			tx.(*transactions.Coinbase).AddReward(*keyPair.PublicKey(), amount)
			return tx, privView
		}

		if err := tx.StandardTx().AddOutput(*addr, amount); err != nil {
			panic(err)
		}

		return tx, privView
	*/
	return nil
}
