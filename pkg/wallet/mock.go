package wallet

import (
	"math/big"
	"math/rand"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/database"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/key"
)

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

func GenerateDecoys(numMixins int) []mlsag.PubKeys {
	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		pubKeyVector := generateDualKey()
		pubKeys = append(pubKeys, pubKeyVector)
	}
	return pubKeys
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

func GenerateInputs(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {

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
