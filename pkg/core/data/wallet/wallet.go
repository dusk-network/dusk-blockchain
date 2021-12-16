// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package wallet

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
)

// ErrSeedFileExists is returned if the seed file already exists.
var ErrSeedFileExists = fmt.Errorf("wallet seed file already exists")

// Wallet encapsulates the wallet.
type Wallet struct {
	db        *database.DB
	netPrefix byte

	// keyPair       *key.Key
	consensusKeys *consensuskey.Keys

	PublicKey keys.PublicKey
	ViewKey   keys.ViewKey
	SecretKey keys.SecretKey
}

// KeysJSON is a struct used to marshal / unmarshal fields to a encrypted file.
type KeysJSON struct {
	Seed         []byte         `json:"seed"`
	SecretKeyBLS []byte         `json:"secret_key_bls"`
	PublicKeyBLS []byte         `json:"public_key_bls"`
	SecretKey    []byte         `json:"secret_key"`
	PublicKey    keys.PublicKey `json:"public_key"`
	ViewKey      keys.ViewKey   `json:"view_key"`
}

// New creates a wallet instance.
func New(seed []byte, netPrefix byte, db *database.DB, password, seedFile string, secretKey *keys.SecretKey) (*Wallet, error) {
	skBuf := new(bytes.Buffer)
	if err := keys.MarshalSecretKey(skBuf, secretKey); err != nil {
		return nil, err
	}

	keysJSON := KeysJSON{
		Seed:      seed,
		SecretKey: skBuf.Bytes(),
	}

	consensusKeys := consensuskey.NewRandKeys()

	keysJSON.SecretKeyBLS = consensusKeys.BLSSecretKey
	keysJSON.PublicKeyBLS = consensusKeys.BLSPubKey

	return LoadFromSeed(netPrefix, db, password, seedFile, keysJSON)
}

// LoadFromSeed loads a wallet from the seed.
func LoadFromSeed(netPrefix byte, db *database.DB, password, seedFile string, keysJSON KeysJSON) (*Wallet, error) {
	secretKey := keys.NewSecretKey()

	err := keys.UnmarshalSecretKey(bytes.NewBuffer(keysJSON.SecretKey), secretKey)
	if err != nil {
		return nil, err
	}

	if secretKey.A == nil || secretKey.B == nil {
		return nil, errors.New("secretKey must be valid")
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
		db:        db,
		netPrefix: netPrefix,
		consensusKeys: &consensuskey.Keys{
			BLSSecretKey: keysJSON.SecretKeyBLS,
			BLSPubKey:    keysJSON.PublicKeyBLS,
		},
		SecretKey: *secretKey,
		PublicKey: keysJSON.PublicKey,
		ViewKey:   keysJSON.ViewKey,
	}

	return w, nil
}

// LoadFromFile loads a wallet from a .dat file.
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

	// secretKey manipulation
	secretKey := keys.NewSecretKey()

	err = keys.UnmarshalSecretKey(bytes.NewBuffer(keysJSON.SecretKey), secretKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:        db,
		netPrefix: netPrefix,
		PublicKey: keysJSON.PublicKey,
		ViewKey:   keysJSON.ViewKey,
		consensusKeys: &consensuskey.Keys{
			BLSPubKey:    keysJSON.PublicKeyBLS,
			BLSSecretKey: keysJSON.SecretKeyBLS,
		},
		SecretKey: *secretKey,
	}, nil
}

// FetchTxHistory will return a slice containing information about all
// transactions made and received with this wallet.
func (w *Wallet) FetchTxHistory() ([]txrecords.TxRecord, error) {
	return w.db.FetchTxRecords()
}

// Keys returns the BLS keys.
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
// (https://github.com/dusk-network/dusk-wallet/blob/master/v2/key/publickey.go#L26).
func (w *Wallet) ToKey(address string) (keys.PublicKey, error) {
	return keys.PublicKey{}, nil
}

// GenerateNewSeed a new seed.
func GenerateNewSeed(Read func(buf []byte) (n int, err error)) ([]byte, error) {
	var seed []byte

	if Read == nil {
		Read = rand.Read
	}

	_, err := Read(seed)
	if err != nil {
		return nil, err
	}

	return seed, nil
}
