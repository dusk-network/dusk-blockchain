package database

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"os"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

const path = "mainnet"

func TestPutGet(t *testing.T) {

	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.Nil(t, err)

	// Close and re-open database
	err = db.Close()
	assert.Nil(t, err)
	db, err = New(path)
	assert.Nil(t, err)

	// Get
	val, err := db.Get(key)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(val, value))

	// Delete
	err = db.Delete(key)
	assert.Nil(t, err)

	// Get after delete
	val, err = db.Get(key)
	assert.Equal(t, leveldb.ErrNotFound, err)
	assert.True(t, bytes.Equal(val, []byte{}))
}

func TestUnlockInputs(t *testing.T) {
	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	input := randInput()
	// This input unlocks at height 1000
	input.unlockHeight = 1000

	// Put it in the DB
	var pubKey ristretto.Point
	pubKey.Rand()
	assert.NoError(t, db.PutInput([]byte{0}, pubKey, input.amount, input.mask, input.privKey, input.unlockHeight))

	// Fetch it and ensure the unlock height is set
	key := append(inputPrefix, pubKey.Bytes()...)
	value, err := db.Get(key)
	assert.NoError(t, err)

	decoded := &inputDB{}
	decoded.Decode(bytes.NewBuffer(value))

	assert.Equal(t, uint64(1000), decoded.unlockHeight)

	// Now run UpdateLockedInputs
	assert.NoError(t, db.UpdateLockedInputs([]byte{0}, 1000))

	value, err = db.Get(key)
	assert.NoError(t, err)

	decoded = &inputDB{}
	decoded.Decode(bytes.NewBuffer(value))

	assert.Equal(t, uint64(0), decoded.unlockHeight)
}

func TestPutFetchTxRecord(t *testing.T) {
	// New
	db, e := New(path)
	if !assert.NoError(t, e) {
		t.FailNow()
	}

	db.UpdateWalletHeight(20)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Create some random txs
	txs := make([]transactions.Transaction, 10)
	privViews := make([]*key.PrivateView, 10)
	for i := range txs {
		tx, privView := randTxForRecord(transactions.TxType(i % 5))
		privViews[i] = privView
		txs[i] = tx
		if err := db.PutTxRecord(tx, txrecords.Direction(i%2), privView); err != nil {
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
			if hex.EncodeToString(tx.StandardTx().Outputs[0].PubKey.P.Bytes()) == record.Recipient {
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
}

func TestClear(t *testing.T) {
	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	db.UpdateWalletHeight(20)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.NoError(t, err)

	// Empty out database
	assert.NoError(t, db.Clear())

	// Info should now be gone entirely
	_, err = db.GetWalletHeight()
	assert.Error(t, err)

	_, err = db.Get([]byte("hello"))
	assert.Error(t, err)
}

func randInput() *inputDB {
	var amount, mask, privKey ristretto.Scalar
	amount.Rand()
	mask.Rand()
	privKey.Rand()
	idb := &inputDB{
		amount:  amount,
		mask:    mask,
		privKey: privKey,
	}

	return idb
}

func randTxForRecord(t transactions.TxType) (transactions.Transaction, *key.PrivateView) {
	var tx transactions.Transaction
	switch t {
	case transactions.StandardType:
		tx, _ = transactions.NewStandard(0, 1, 100)
	case transactions.TimelockType:
		tx, _ = transactions.NewTimelock(0, 1, 100, 10000)
	case transactions.BidType:
		tx, _ = transactions.NewBid(0, 1, 100, 5000, make([]byte, 32))
	case transactions.StakeType:
		tx, _ = transactions.NewStake(0, 1, 100, 2130, make([]byte, 32), make([]byte, 129))
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
}
