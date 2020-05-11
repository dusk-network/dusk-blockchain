package wallet

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
	"golang.org/x/crypto/sha3"
)

// DUSK is one whole unit of DUSK.
const DUSK = uint64(100000000)

// ErrSeedFileExists is returned if the seed file already exists
var ErrSeedFileExists = fmt.Errorf("wallet seed file already exists")

// Wallet encapsulates the wallet
type Wallet struct {
	db        *database.DB
	netPrefix byte

	//keyPair       *key.Key
	consensusKeys *consensuskey.Keys

	PublicKey transactions.PublicKey
	ViewKey   transactions.ViewKey
	SecretKey transactions.SecretKey
}

// New creates a wallet instance
func New(Read func(buf []byte) (n int, err error), seed []byte, netPrefix byte, db *database.DB, password string, seedFile, secretFile string, secretKey *transactions.SecretKey) (*Wallet, error) {

	//create new seed if seed comes empty
	if len(seed) == 0 {
		var err error
		seed, err = GenerateNewSeed(Read)
		if err != nil {
			return nil, err
		}
	}

	return LoadFromSeed(seed, netPrefix, db, password, seedFile, secretFile, secretKey)
}

// GenerateNewSeed a new seed
func GenerateNewSeed(Read func(buf []byte) (n int, err error)) ([]byte, error) {
	var seed []byte
	if Read == nil {
		Read = rand.Read
	}

	for {
		// random seed
		seed = make([]byte, 64)
		_, err := Read(seed)
		if err != nil {
			return nil, err
		}

		// Ensure the seed can be used for generating a BLS keypair.
		_, err = generateKeys(seed)
		if err == nil {
			break
		}

		if err != io.EOF {
			return nil, err
		}
		// If not, we retry.
	}
	return seed, nil
}

// LoadFromSeed loads a wallet from the seed
func LoadFromSeed(seed []byte, netPrefix byte, db *database.DB, password string, seedFile, secretFile string, secretKey *transactions.SecretKey) (*Wallet, error) {
	if len(seed) < 64 {
		return nil, errors.New("seed must be atleast 64 bytes in size")
	}

	if secretKey.A == nil || secretKey.B == nil {
		return nil, errors.New("secretKey must be valid")
	}

	//TODO: are we sure we want to save the seed when loading ?
	if err := saveEncrypted(seed, password, seedFile); err != nil {
		return nil, err
	}

	//TODO: are we sure we want to save the secretKey when loading ?
	//secretKey manipulation
	skBuf := new(bytes.Buffer)
	if err := transactions.MarshalSecretKey(skBuf, *secretKey); err != nil {
		return nil, err
	}

	if err := saveEncrypted(skBuf.Bytes(), password, secretFile); err != nil {
		return nil, err
	}

	consensusKeys, kerr := generateKeys(seed)
	if kerr != nil {
		return nil, kerr
	}

	//TODO: KEYS generate PublicKey and ViewKey from SecretKey
	w := &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		consensusKeys: &consensusKeys,
		SecretKey:     *secretKey,
		PublicKey:     transactions.PublicKey{}, //TODO: KEYS
		ViewKey:       transactions.ViewKey{},   //TODO: KEYS
	}

	return w, nil
}

// LoadFromFile loads a wallet from a .dat file
func LoadFromFile(netPrefix byte, db *database.DB, password string, seedFile, secretKeyFile string) (*Wallet, error) {

	seed, err := fetchEncrypted(password, seedFile)
	if err != nil {
		return nil, err
	}

	secretKeyByteArr, err := fetchEncrypted(password, secretKeyFile)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateKeys(seed)
	if err != nil {
		return nil, err
	}

	// secretKey manipulation
	secretKey := new(transactions.SecretKey)
	err = transactions.UnmarshalSecretKey(bytes.NewBuffer(secretKeyByteArr), secretKey)
	if err != nil {
		return nil, err
	}

	// TODO: publicKey manipulation
	// TODO: viewKey manipulation

	return &Wallet{
		db:        db,
		netPrefix: netPrefix,
		//keyPair:       key.NewKeyPair(seed),
		PublicKey:     transactions.PublicKey{}, // TODO: public key should be saved and retrieved later to spare a roundtrip to rusk
		ViewKey:       transactions.ViewKey{},   // TODO: view key should be saved and retrieved later to spare a roundtrip to rusk
		consensusKeys: &consensusKeys,
		SecretKey:     *secretKey,
	}, nil
}

// FetchTxHistory will return a slice containing information about all
// transactions made and received with this wallet.
func (w *Wallet) FetchTxHistory() ([]txrecords.TxRecord, error) {
	return w.db.FetchTxRecords()
}

// Keys returns the BLS keys
func (w *Wallet) Keys() consensuskey.Keys {
	return *w.consensusKeys
}

// ClearDatabase will remove all info from the database.
func (w *Wallet) ClearDatabase() error {
	return w.db.Clear()
}

// ToKey gets a string as public address and returns a PublicKey
// FIXME: this is used within the cmd/wallet transferDusk function. Not clear
// if still needed.
// Old implementation can be find here
// (https://github.com/dusk-network/dusk-wallet/blob/master/v2/key/publickey.go#L26)
func (w *Wallet) ToKey(address string) (transactions.PublicKey, error) {
	return transactions.PublicKey{}, nil
}

func generateKeys(seed []byte) (consensuskey.Keys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	return consensuskey.NewKeysFromBytes(consensusSeed)
}
