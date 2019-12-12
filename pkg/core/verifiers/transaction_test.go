package verifiers_test

import (
	"crypto/rand"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/block"
	walletdb "github.com/dusk-network/dusk-wallet/database"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/dusk-network/dusk-wallet/wallet"
	"github.com/stretchr/testify/assert"
)

// Test that verifying a transaction with locked inputs throws an error.
func TestLockedInputs(t *testing.T) {
	// Ensure we use the heavy driver for this test
	r := config.Registry{}
	r.Database.Driver = heavy.DriverName
	r.Database.Dir = "db"
	r.General.Network = "testnet"
	config.Mock(&r)

	// Make sure we clean up
	defer os.RemoveAll(config.Get().Database.Dir)

	// Create a wallet with mock functions
	aliceDB, err := walletdb.New("alice")
	assert.NoError(t, err)
	alice, err := wallet.New(rand.Read, 2, aliceDB, wallet.GenerateDecoys, wallet.GenerateInputs, "pass", "alice.dat")
	assert.NoError(t, err)

	// Ensure clean up
	defer os.RemoveAll("alice")
	defer os.Remove("alice.dat")

	// Credit alice with a standard output first, so that we have a decoy
	var amount ristretto.Scalar
	amount.SetBigInt(big.NewInt(10))
	tx, err := alice.NewStandardTx(100)
	assert.NoError(t, err)
	alicePubAddr, err := alice.PublicAddress()
	assert.NoError(t, err)
	tx.AddOutput(key.PublicAddress(alicePubAddr), amount)
	err = alice.Sign(tx)
	assert.NoError(t, err)
	// Rip out the change to avoid random failures
	tx.Outputs = tx.Outputs[0:1]
	// Database setup for test
	_, db := heavy.CreateDBConnection()
	blk := writeTxToDatabase(t, db, tx, 0)
	_, _, err = alice.CheckWireBlock(*blk)
	assert.NoError(t, err)

	// Create stake tx
	amount.SetBigInt(big.NewInt(10000000))
	stake, err := alice.NewStakeTx(100, 10000, amount)
	assert.NoError(t, err)
	err = alice.Sign(stake)
	assert.NoError(t, err)
	// Rip out the change output so that the test will not randomly fail
	stake.Outputs = stake.Outputs[0:1]

	blk = writeTxToDatabase(t, db, stake, 1)
	_, _, err = alice.CheckWireBlock(*blk)
	assert.NoError(t, err)

	// Unlock the output for the wallet, so we can use it in the next tx
	privSpend, err := alice.PrivateSpend()
	assert.NoError(t, err)
	aliceDB.UpdateLockedInputs(privSpend, 10001)

	// Now, set our FetchInputs function to get inputs from the db
	alice, err = wallet.LoadFromFile(2, aliceDB, fetchDecoys, fetchInputs, "pass", "alice.dat")

	// Create a standard tx, using the locked output that we sent to alice
	tx, err = alice.NewStandardTx(100)
	assert.NoError(t, err)
	amount.SetBigInt(big.NewInt(1000000))
	tx.AddOutput(key.PublicAddress("pippo"), amount)
	err = alice.Sign(tx)
	assert.NoError(t, err)

	// Checking the tx should fail with a specific error
	assert.Equal(t, "transaction contains one or more locked inputs", verifiers.CheckTx(db, 0, uint64(time.Now().Unix()), tx).Error())
}

// Write a block with one transaction to the db.
func writeTxToDatabase(t *testing.T, db database.DB, tx transactions.Transaction, height uint64) *block.Block {
	blk := block.NewBlock()
	blk.Header.Height = height
	blk.Header.Version = 0
	blk.Header.Timestamp = time.Now().Unix()
	blk.Header.Hash = make([]byte, 32)
	blk.Header.Seed = make([]byte, 33)
	blk.Header.PrevBlockHash = make([]byte, 32)
	blk.Header.TxRoot = make([]byte, 32)

	blk.AddTx(tx)
	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	}))

	return blk
}

func fetchDecoys(numMixins int) []mlsag.PubKeys {
	_, db := heavy.CreateDBConnection()

	var pubKeys []mlsag.PubKeys
	var decoys []ristretto.Point
	db.View(func(t database.Transaction) error {
		decoys = t.FetchDecoys(numMixins)
		return nil
	})

	// Potential panic if the database does not have enough decoys
	for i := 0; i < numMixins; i++ {
		var keyVector mlsag.PubKeys
		keyVector.AddPubKey(decoys[0])

		var secondaryKey ristretto.Point
		secondaryKey.Rand()
		keyVector.AddPubKey(secondaryKey)

		pubKeys = append(pubKeys, keyVector)
	}
	return pubKeys
}

func fetchInputs(netPrefix byte, db *walletdb.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}
