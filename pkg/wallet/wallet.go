package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"
	"golang.org/x/crypto/sha3"

	"github.com/bwesterb/go-ristretto"
	"github.com/syndtr/goleveldb/leveldb"
)

// Number of mixins per ring. ringsize = mixin + 1
const numMixins = 7

// FetchInputs returns a slice of inputs such that Sum(Inputs)- Sum(Outputs) >= 0
// If > 0, then a change address is created for the remaining amount
type FetchInputs func(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error)

type Wallet struct {
	db            *database.DB
	netPrefix     byte
	keyPair       *key.Key
	consensusKeys *user.Keys
	fetchDecoys   transactions.FetchDecoys
	fetchInputs   FetchInputs
}

func New(netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string) (*Wallet, error) {

	// random seed
	seed := make([]byte, 64)
	_, err := rand.Read(seed)
	if err != nil {
		return nil, err
	}

	err = saveSeed(seed, password)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateConsensusKeys(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		fetchDecoys:   fDecoys,
		fetchInputs:   fInputs,
	}, nil
}

func Load(netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string) (*Wallet, error) {

	seed, err := fetchSeed(password)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateConsensusKeys(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		fetchDecoys:   fDecoys,
		fetchInputs:   fInputs,
	}, nil
}

func (w *Wallet) NewStandardTx(fee int64) (*transactions.StandardTx, error) {
	tx, err := transactions.NewStandard(w.netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewCoinbaseTx() *transactions.CoinbaseTx {
	tx := transactions.NewCoinBaseTx(w.netPrefix)
	return tx
}

// CheckWireBlockSpent checks if the block has any outputs spent by this wallet
func (w *Wallet) CheckWireBlockSpent(blk block.Block) (uint64, error) {

	var totalSpentCount uint64

	txInCheckers := NewTxInChecker(blk)

	for _, txchecker := range txInCheckers {
		spentCount, err := w.scanInputs(txchecker)
		if err != nil {
			return spentCount, err
		}
		totalSpentCount += spentCount
	}

	return totalSpentCount, nil
}

func (w *Wallet) scanInputs(txChecker TxInChecker) (uint64, error) {

	var spentCount uint64

	for _, keyImage := range txChecker.keyImages {
		pubKey, err := w.db.Get(keyImage)
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			return spentCount, err
		}

		spentCount++

		err = w.db.RemoveInput(pubKey)
		if err != nil {
			return spentCount, err
		}
	}
	return spentCount, nil
}

// CheckWireBlockReceived checks if the wire block has transactions for this wallet
func (w *Wallet) CheckWireBlockReceived(blk block.Block) (uint64, error) {

	var totalReceivedCount uint64

	txCheckers := NewTxOutChecker(blk)
	for i, txchecker := range txCheckers {
		if i == 0 {
			// coinbase tx
			receivedCount, err := w.scanOutputs(false, txchecker)
			if err != nil {
				return receivedCount, err
			}
			totalReceivedCount += receivedCount
			continue
		}
		receivedCount, err := w.scanOutputs(true, txchecker)
		if err != nil {
			return receivedCount, err
		}
		totalReceivedCount += receivedCount
	}

	return totalReceivedCount, nil
}

func (w *Wallet) scanOutputs(valuesEncrypted bool, txchecker TxOutChecker) (uint64, error) {

	privView, err := w.keyPair.PrivateView()
	if err != nil {
		return 0, err
	}
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, err
	}

	var receiveCount uint64

	for i, output := range txchecker.Outputs {
		privKey, ok := w.keyPair.DidReceiveTx(txchecker.R, output.PubKey, uint32(i))
		if !ok {
			continue
		}

		receiveCount++

		var amount, mask ristretto.Scalar
		amount.Set(&output.EncryptedAmount)
		mask.Set(&output.EncryptedMask)

		if valuesEncrypted {
			amount = transactions.DecryptAmount(output.EncryptedAmount, txchecker.R, uint32(i), *privView)
			mask = transactions.DecryptMask(output.EncryptedMask, txchecker.R, uint32(i), *privView)
		}

		err := w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, *privKey)
		if err != nil {
			return receiveCount, err
		}

		// cache the keyImage, so we can quickly check whether our input was spent
		var pubKey ristretto.Point
		pubKey.ScalarMultBase(privKey)
		keyImage := mlsag.CalculateKeyImage(*privKey, pubKey)

		err = w.db.Put(keyImage.Bytes(), output.PubKey.P.Bytes())
		if err != nil {
			return receiveCount, err
		}
	}

	return receiveCount, nil
}

// AddInputs adds up the total outputs and fee then fetches inputs to consolidate this
func (w *Wallet) AddInputs(tx *transactions.StandardTx) error {

	totalAmount := tx.Fee.BigInt().Int64() + tx.TotalSent.BigInt().Int64()

	inputs, changeAmount, err := w.fetchInputs(w.netPrefix, w.db, totalAmount, w.keyPair)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		err := tx.AddInput(input)
		if err != nil {
			return err
		}
	}

	changeAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return err
	}

	// Convert int64 to ristretto value
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(changeAmount))

	return tx.AddOutput(*changeAddr, x)
}

func (w *Wallet) Sign(tx *transactions.StandardTx) error {

	// Assuming user has added all of the outputs

	// Fetch Inputs
	err := w.AddInputs(tx)
	if err != nil {
		return err
	}

	// Fetch decoys
	err = tx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	return tx.Prove()
}

func (w *Wallet) PublicKey() key.PublicKey {
	return *w.keyPair.PublicKey()
}

// Save saves the seed to a dat file
func saveSeed(seed []byte, password string) error {

	digest := sha3.Sum256([]byte(password))

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	return ioutil.WriteFile("wallet.dat", gcm.Seal(nonce, nonce, seed, nil), 0777)
}

//Modified from https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
func fetchSeed(password string) ([]byte, error) {

	digest := sha3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile("wallet.dat")
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	seed, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return seed, nil
}

func generateConsensusKeys(seed []byte) (user.Keys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	reader := bytes.NewReader(consensusSeed)

	return user.NewKeysFromSeed(reader)
}
