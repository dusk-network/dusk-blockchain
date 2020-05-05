package wallet

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"reflect"

	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
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

	keyPair       *key.Key
	consensusKeys *consensuskey.Keys

	secretKey *rusk.SecretKey
}

// New creates a wallet instance
func New(Read func(buf []byte) (n int, err error), seed []byte, netPrefix byte, db *database.DB, password string, seedFile, secretFile string, secretKey *rusk.SecretKey) (*Wallet, error) {

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
func LoadFromSeed(seed []byte, netPrefix byte, db *database.DB, password string, seedFile, secretFile string, secretKey *rusk.SecretKey) (*Wallet, error) {
	if len(seed) < 64 {
		return nil, errors.New("seed must be atleast 64 bytes in size")
	}

	if secretKey.A == nil || secretKey.B == nil {
		return nil, errors.New("secretKey must be valid")
	}

	//TODO: are we sure we want to save the seed when loading ?
	err := saveEncrypted(seed, password, seedFile)
	if err != nil {
		return nil, err
	}

	//TODO: are we sure we want to save the secretKey when loading ?
	//secretKey manipulation
	secretKeyArr, err := proto.Marshal(secretKey)
	if err != nil {
		return nil, err
	}

	secretKeyNew := new(rusk.SecretKey)
	err = proto.Unmarshal(secretKeyArr, secretKeyNew)
	if err != nil {
		return nil, err
	}

	//TODO: remove this once we are confident this is really working
	if !reflect.DeepEqual(secretKey.A.Data, secretKeyNew.A.Data) || !reflect.DeepEqual(secretKey.B.Data, secretKeyNew.B.Data) {
		log.WithField("secretKey.A.Data", secretKey.A.Data).
			WithField("secretKeyNew.A.Data", secretKeyNew.A.Data).
			WithField("secretKey.B.Data", secretKey.B.Data).
			WithField("secretKeyNew.B.Data", secretKeyNew.B.Data).
			//WithField("secretKey",secretKey.String()).
			//WithField("secretKeyNew",secretKeyNew.String()).
			//WithField("string eq", secretKey.String() == secretKeyNew.String()).
			Error("karaidiasa")
		panic("karaidiasa")
	}

	err = saveEncrypted(secretKeyArr, password, secretFile)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateKeys(seed)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		secretKey:     secretKey,
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
	secretKey := new(rusk.SecretKey)
	err = proto.Unmarshal(secretKeyByteArr, secretKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		secretKey:     secretKey,
	}, nil
}

// FetchTxHistory will return a slice containing information about all
// transactions made and received with this wallet.
func (w *Wallet) FetchTxHistory() ([]txrecords.TxRecord, error) {
	return w.db.FetchTxRecords()
}

// PublicKey returns the wallet public key
func (w *Wallet) PublicKey() key.PublicKey {
	return *w.keyPair.PublicKey()
}

// SecretKey returns the wallet secret key
func (w *Wallet) SecretKey() *rusk.SecretKey {
	return w.secretKey
}

// Keys returns the BLS keys
func (w *Wallet) Keys() consensuskey.Keys {
	return *w.consensusKeys
}

// ClearDatabase will remove all info from the database.
func (w *Wallet) ClearDatabase() error {
	return w.db.Clear()
}

func generateKeys(seed []byte) (consensuskey.Keys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	return consensuskey.NewKeysFromBytes(consensusSeed)
}
