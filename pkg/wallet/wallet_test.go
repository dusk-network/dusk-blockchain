package wallet

import (
	"bytes"
	"math/big"
	"os"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"

	"math/rand"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

var dbPath = "testDb"

func TestNewWallet(t *testing.T) {
	netPrefix := byte(1)

	db, err := database.New(dbPath)
	assert.Nil(t, err)
	defer os.RemoveAll(dbPath)

	w, err := New(randReader, netPrefix, db, generateDecoys, fetchInputs, "pass")
	assert.Nil(t, err)

	// wrong wallet password
	loadedWallet, err := LoadFromFile(netPrefix, db, generateDecoys, fetchInputs, "wrongPass")
	assert.NotNil(t, err)

	// correct wallet password
	loadedWallet, err = LoadFromFile(netPrefix, db, generateDecoys, fetchInputs, "pass")
	assert.Nil(t, err)

	assert.Equal(t, w.PublicKey(), loadedWallet.PublicKey())

	assert.Equal(t, w.consensusKeys.EdSecretKey, loadedWallet.consensusKeys.EdSecretKey)
	assert.Equal(t, w.consensusKeys.BLSSecretKey, loadedWallet.consensusKeys.BLSSecretKey)
	assert.True(t, bytes.Equal(w.consensusKeys.BLSPubKeyBytes, loadedWallet.consensusKeys.BLSPubKeyBytes))

}

func TestReceivedTx(t *testing.T) {
	netPrefix := byte(1)
	fee := int64(0)

	db, err := database.New(dbPath)
	assert.Nil(t, err)
	defer os.RemoveAll(dbPath)

	w, err := New(randReader, netPrefix, db, generateDecoys, fetchInputs, "pass")
	assert.Nil(t, err)

	tx, err := w.NewStandardTx(fee)
	assert.Nil(t, err)

	var tenDusk ristretto.Scalar
	tenDusk.SetBigInt(big.NewInt(10))

	sendersAddr := generateSendAddr(t, netPrefix, w.keyPair)
	assert.Nil(t, err)

	err = tx.AddOutput(sendersAddr, tenDusk)
	assert.Nil(t, err)

	err = w.Sign(tx)
	assert.Nil(t, err)

	for _, output := range tx.Outputs {
		_, ok := w.keyPair.DidReceiveTx(tx.R, output.PubKey, output.Index)
		assert.True(t, ok)
	}

	var destKeys []ristretto.Point
	for _, output := range tx.Outputs {
		destKeys = append(destKeys, output.PubKey.P)
	}
	assert.False(t, hasDuplicates(destKeys))
}

func TestCheckBlock(t *testing.T) {
	netPrefix := byte(1)

	alice := generateWallet(t, netPrefix, "alice")
	bob := generateWallet(t, netPrefix, "bob")
	bobAddr, err := bob.keyPair.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)

	var numTxs = 3 // numTxs to send to Bob

	var blk block.Block
	for i := 0; i < numTxs; i++ {
		tx := generateStandardTx(t, *bobAddr, 20, alice)
		wireStandardTx, err := tx.WireStandardTx()
		assert.Nil(t, err)
		blk.AddTx(wireStandardTx)
	}

	count, err := bob.CheckWireBlockReceived(blk)
	assert.Nil(t, err)
	assert.Equal(t, uint64(numTxs), count)

	_, err = alice.CheckWireBlockSpent(blk)
	assert.Nil(t, err)
}

func generateWallet(t *testing.T, netPrefix byte, path string) *Wallet {

	db, err := database.New(path)
	assert.Nil(t, err)
	defer os.RemoveAll(path)

	w, err := New(randReader, netPrefix, db, generateDecoys, fetchInputs, "pass")
	assert.Nil(t, err)
	return w
}

func generateStandardTx(t *testing.T, receiver key.PublicAddress, amount int64, sender *Wallet) *transactions.StandardTx {
	tx, err := sender.NewStandardTx(0)
	assert.Nil(t, err)

	var duskAmount ristretto.Scalar
	duskAmount.SetBigInt(big.NewInt(amount))

	err = tx.AddOutput(receiver, duskAmount)
	assert.Nil(t, err)

	err = sender.Sign(tx)
	assert.Nil(t, err)

	return tx
}

func fetchInputs(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {

	// This function shoud store the inputs in a database
	// Upon calling fetchInputs, we use the keyPair to get the privateKey from the
	// one time pubkey

	var inputs []*transactions.Input
	numInputs := 7

	addresses, R := generateOutputAddress(key, netPrefix, numInputs)

	rand.Seed(time.Now().Unix())
	randAmount := rand.Int63n(totalAmount) + int64(totalAmount)/int64(numInputs) + 1 // [totalAmount/4 + 1, totalAmount*4]
	remainder := (randAmount * int64(numInputs)) - totalAmount
	if remainder < 0 {
		remainder = 0
	}

	for index, addr := range addresses {
		var amount, mask ristretto.Scalar
		amount.SetBigInt(big.NewInt(randAmount))
		mask.Rand()

		// Fetch the privKey for each addresses
		privKey, _ := key.DidReceiveTx(R, *addr, uint32(index))

		input := transactions.NewInput(amount, mask, *privKey)
		inputs = append(inputs, input)
	}

	return inputs, remainder, nil
}

func generateSendAddr(t *testing.T, netPrefix byte, randKeyPair *key.Key) key.PublicAddress {
	pubAddr, err := randKeyPair.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)
	return *pubAddr
}

func generateOutputAddress(keyPair *key.Key, netPrefix byte, num int) ([]*key.StealthAddress, ristretto.Point) {
	var res []*key.StealthAddress

	sendersPubKey := keyPair.PublicKey()

	var r ristretto.Scalar
	r.Rand()

	var R ristretto.Point
	R.ScalarMultBase(&r)

	for i := 0; i < num; i++ {
		stealthAddr := sendersPubKey.StealthAddress(r, uint32(i))
		res = append(res, stealthAddr)
	}
	return res, R
}

// https://www.dotnetperls.com/duplicates-go
func hasDuplicates(elements []ristretto.Point) bool {
	encountered := map[ristretto.Point]bool{}

	for v := range elements {
		if encountered[elements[v]] == true {
			return true
		}
		encountered[elements[v]] = true
	}
	return false
}

func sliceToPoint(t *testing.T, b []byte) ristretto.Point {
	if len(b) != 32 {
		t.Fatal("slice to point must be given a 32 byte slice")
	}
	var c ristretto.Point
	var byts [32]byte
	copy(byts[:], b)
	c.SetBytes(&byts)
	return c
}

func generateDualKey() mlsag.PubKeys {
	pubkeys := mlsag.PubKeys{}

	var primaryKey ristretto.Point
	primaryKey.Rand()
	pubkeys.AddPubKey(primaryKey)

	var secondaryKey ristretto.Point
	secondaryKey.Rand()
	pubkeys.AddPubKey(secondaryKey)

	return pubkeys
}

func generateDecoys(numMixins int) []mlsag.PubKeys {
	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		pubKeyVector := generateDualKey()
		pubKeys = append(pubKeys, pubKeyVector)
	}
	return pubKeys
}

func randReader(b []byte) (n int, err error) {
	return len(b), nil
}
