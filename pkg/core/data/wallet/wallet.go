package wallet

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
	"golang.org/x/crypto/sha3"
)

// DUSK is one whole unit of DUSK.
const DUSK = uint64(10000000000)

// ErrSeedFileExists is returned if the seed file already exists
var ErrSeedFileExists = fmt.Errorf("wallet seed file already exists")

// Wallet encapsulates the wallet
type Wallet struct {
	db        *database.DB
	netPrefix byte

	//keyPair       *key.Key
	consensusKeys *consensuskey.Keys

	PublicKey keys.PublicKey
	ViewKey   keys.ViewKey
	SecretKey keys.SecretKey
}

// KeysJSON is a struct used to marshal / unmarshal fields to a encrypted file
type KeysJSON struct {
	Seed      []byte         `json:"seed"`
	SecretKey []byte         `json:"secret_key"`
	PublicKey keys.PublicKey `json:"public_key"`
	ViewKey   keys.ViewKey   `json:"view_key"`
}

// New creates a wallet instance
func New(Read func(buf []byte) (n int, err error), seed []byte, netPrefix byte, db *database.DB, password, seedFile string, secretKey *keys.SecretKey) (*Wallet, error) {
	//create new seed if seed comes empty
	if len(seed) == 0 {
		var err error
		seed, err = GenerateNewSeed(Read)
		if err != nil {
			return nil, err
		}
	}

	skBuf := new(bytes.Buffer)
	if err := keys.MarshalSecretKey(skBuf, secretKey); err != nil {
		return nil, err
	}

	keysJSON := KeysJSON{
		Seed:      seed,
		SecretKey: skBuf.Bytes(),
	}
	return LoadFromSeed(netPrefix, db, password, seedFile, keysJSON)
}

// LoadFromSeed loads a wallet from the seed
func LoadFromSeed(netPrefix byte, db *database.DB, password, seedFile string, keysJSON KeysJSON) (*Wallet, error) {
	if len(keysJSON.Seed) < 64 {
		return nil, errors.New("seed must be at least 64 bytes in size")
	}

	secretKey := keys.NewSecretKey()
	err := keys.UnmarshalSecretKey(bytes.NewBuffer(keysJSON.SecretKey), secretKey)
	if err != nil {
		return nil, err
	}

	if secretKey.A == nil || secretKey.B == nil {
		return nil, errors.New("secretKey must be valid")
	}

	consensusKeys, err := generateKeys(keysJSON.Seed)
	if err != nil {
		return nil, err
	}

	// transform keysJSON to []byte
	data, err := json.Marshal(keysJSON)
	if err != nil {
		return nil, err
	}

	// store it in a encrypted file
	if err := saveEncrypted(data, password, seedFile); err != nil {
		return nil, err
	}

	w := &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		consensusKeys: &consensusKeys,
		SecretKey:     *secretKey,
		PublicKey:     keysJSON.PublicKey,
		ViewKey:       keysJSON.ViewKey,
	}

	return w, nil
}

// LoadFromFile loads a wallet from a .dat file
func LoadFromFile(netPrefix byte, db *database.DB, password string, seedFile string) (*Wallet, error) {

	keysJSONArr, err := fetchEncrypted(password, seedFile)
	if err != nil {
		return nil, err
	}

	// transform []byte to keysJSON
	keysJSON := new(KeysJSON)
	err = json.Unmarshal(keysJSONArr, keysJSON)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateKeys(keysJSON.Seed)
	if err != nil {
		return nil, err
	}

	// secretKey manipulation
	secretKey := keys.NewSecretKey()
	err = keys.UnmarshalSecretKey(bytes.NewBuffer(keysJSON.SecretKey), secretKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		PublicKey:     keysJSON.PublicKey,
		ViewKey:       keysJSON.ViewKey,
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
func (w *Wallet) ToKey(address string) (keys.PublicKey, error) {
	return keys.PublicKey{}, nil
}

func generateKeys(seed []byte) (consensuskey.Keys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	return consensuskey.NewKeysFromBytes(consensusSeed)
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
