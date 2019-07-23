package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"math/big"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"
	"gitlab.dusk.network/dusk-core/zkproof"
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
	db        *database.DB
	netPrefix byte

	keyPair       *key.Key
	consensusKeys *user.Keys

	fetchDecoys transactions.FetchDecoys
	fetchInputs FetchInputs
}

type SignableTx interface {
	AddDecoys(numMixins int, f transactions.FetchDecoys) error
	Prove() error
	Standard() (*transactions.StandardTx, error)
}

func New(Read func(buf []byte) (n int, err error), netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string) (*Wallet, error) {

	// random seed
	seed := make([]byte, 64)
	_, err := Read(seed)
	if err != nil {
		return nil, err
	}
	return LoadFromSeed(seed, netPrefix, db, fDecoys, fInputs, password)
}

func LoadFromSeed(seed []byte, netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string) (*Wallet, error) {
	if len(seed) < 64 {
		return nil, errors.New("seed must be atleast 64 bytes in size")
	}
	err := saveSeed(seed, password)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateConsensusKeys(seed)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		fetchDecoys:   fDecoys,
		fetchInputs:   fInputs,
	}

	// Check if this is a new wallet
	_, err = w.db.GetWalletHeight()
	if err == nil {
		return w, nil
	}

	if err != leveldb.ErrNotFound {
		return nil, err
	}

	// Add height of zero into database
	err = w.UpdateWalletHeight(0)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func LoadFromFile(netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string) (*Wallet, error) {

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

func (w *Wallet) NewStakeTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.StakeTx, error) {
	edPubBytes := w.consensusKeys.EdPubKeyBytes
	blsPubBytes := w.consensusKeys.BLSPubKeyBytes
	tx, err := transactions.NewStakeTx(w.netPrefix, fee, lockTime, edPubBytes, blsPubBytes)
	if err != nil {
		return nil, err
	}

	// Send locked stake amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewBidTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.BidTx, error) {
	privateSpend, err := w.keyPair.PrivateSpend()
	privateSpend.Bytes()

	// TODO: index is currently set to be zero.
	// To avoid any privacy implications, the wallet should increment
	// the index by how many bidding txs are seen
	mBytes := generateM(privateSpend.Bytes(), 0)
	tx, err := transactions.NewBidTx(w.netPrefix, fee, lockTime, mBytes)
	if err != nil {
		return nil, err
	}

	// Send bid amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewCoinbaseTx() *transactions.CoinbaseTx {
	tx := transactions.NewCoinBaseTx(w.netPrefix)
	return tx
}

func (w *Wallet) CheckWireBlock(blk block.Block) (uint64, uint64, error) {

	spentCount, err := w.CheckWireBlockSpent(blk)
	if err != nil {
		return 0, 0, err
	}

	receivedCount, err := w.CheckWireBlockReceived(blk)
	if err != nil {
		return 0, 0, err
	}

	err = w.UpdateWalletHeight(blk.Header.Height + 1)
	if err != nil {
		return 0, 0, err
	}
	return spentCount, receivedCount, nil
}

// CheckWireBlockSpent checks if the block has any outputs spent by this wallet
// Returns the number of txs that the sender spent funds in
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

	var didSpendFunds uint64

	for _, keyImage := range txChecker.keyImages {
		pubKey, err := w.db.Get(keyImage)
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			return didSpendFunds, err
		}

		err = w.db.RemoveInput(pubKey)
		if err != nil {
			return didSpendFunds, err
		}
	}
	return didSpendFunds, nil
}

// CheckWireBlockReceived checks if the wire block has transactions for this wallet
// Returns the number of tx's that the reciever recieved funds in
func (w *Wallet) CheckWireBlockReceived(blk block.Block) (uint64, error) {

	var totalReceivedCount uint64

	txCheckers := NewTxOutChecker(blk)
	for _, txchecker := range txCheckers {
		receivedCount, err := w.scanOutputs(txchecker)
		if err != nil {
			return receivedCount, err
		}
		totalReceivedCount += receivedCount
	}

	return totalReceivedCount, nil
}

// scans the outputs of one transaction
func (w *Wallet) scanOutputs(txchecker TxOutChecker) (uint64, error) {

	privView, err := w.keyPair.PrivateView()
	if err != nil {
		return 0, err
	}
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, err
	}

	var didReceiveFunds uint64

	for i, output := range txchecker.Outputs {
		privKey, ok := w.keyPair.DidReceiveTx(txchecker.R, output.PubKey, uint32(i))
		if !ok {
			continue
		}

		didReceiveFunds = 1

		var amount, mask ristretto.Scalar
		amount.Set(&output.EncryptedAmount)
		mask.Set(&output.EncryptedMask)

		if txchecker.encryptedValues {
			amount = transactions.DecryptAmount(output.EncryptedAmount, txchecker.R, uint32(i), *privView)
			mask = transactions.DecryptMask(output.EncryptedMask, txchecker.R, uint32(i), *privView)
		}

		err := w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, *privKey)
		if err != nil {
			return didReceiveFunds, err
		}

		// cache the keyImage, so we can quickly check whether our input was spent
		var pubKey ristretto.Point
		pubKey.ScalarMultBase(privKey)
		keyImage := mlsag.CalculateKeyImage(*privKey, pubKey)

		err = w.db.Put(keyImage.Bytes(), output.PubKey.P.Bytes())
		if err != nil {
			return didReceiveFunds, err
		}
	}

	return didReceiveFunds, nil
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

func (w *Wallet) Sign(tx SignableTx) error {

	// Assuming user has added all of the outputs

	standardTx, err := tx.Standard()
	if err != nil {
		return err
	}

	// Fetch Inputs
	err = w.AddInputs(standardTx)
	if err != nil {
		return err
	}

	// Fetch decoys
	err = standardTx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	return tx.Prove()
}

func (w *Wallet) Balance() (float64, error) {
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, err
	}
	balanceInt, err := w.db.FetchBalance(privSpend.Bytes())
	return float64(balanceInt / cfg.DUSK), nil
}

func (w *Wallet) GetSavedHeight() (uint64, error) {
	return w.db.GetWalletHeight()
}

func (w *Wallet) UpdateWalletHeight(newHeight uint64) error {
	return w.db.UpdateWalletHeight(newHeight)
}

func (w *Wallet) PublicKey() key.PublicKey {
	return *w.keyPair.PublicKey()
}

func (w *Wallet) PublicAddress() (string, error) {
	pubAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return "", err
	}
	return pubAddr.String(), nil
}

func (w *Wallet) ConsensusKeys() user.Keys {
	return *w.consensusKeys
}

func (w *Wallet) PrivateSpend() ([]byte, error) {
	privateSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return nil, err
	}

	return privateSpend.Bytes(), nil
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

	return ioutil.WriteFile(cfg.Get().Wallet.File, gcm.Seal(nonce, nonce, seed, nil), 0777)
}

//Modified from https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
func fetchSeed(password string) ([]byte, error) {

	digest := sha3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile(cfg.Get().Wallet.File)
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

	return user.NewKeysFromBytes(consensusSeed)
}

func generateM(PrivateSpend []byte, index uint32) []byte {

	// To make K deterministic
	// We will calculate K = PrivateSpend || Index
	// Index is the number of Bidding transactions that has
	// been initiated. This information should be available to the wallet
	// M = H(K)

	numBidTxsSeen := make([]byte, 4)
	binary.BigEndian.PutUint32(numBidTxsSeen, index)

	KBytes := append(PrivateSpend, numBidTxsSeen...)

	// Encode K as a ristretto Scalar
	var k ristretto.Scalar
	k.Derive(KBytes)

	m := zkproof.CalculateM(k)
	return m.Bytes()
}
