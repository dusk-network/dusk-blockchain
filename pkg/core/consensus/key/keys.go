// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package key

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"golang.org/x/crypto/sha3"
)

// ErrAlreadyExists file already exists.
var ErrAlreadyExists = errors.New("file already exists")

// Keys are the keys used during consensus.
type Keys struct {
	BLSPubKey    []byte
	BLSSecretKey []byte
}

// NewRandKeys creates a new pair of BLS keys.
func NewRandKeys() Keys {
	sk, pk := bls.GenerateKeys()

	return Keys{
		BLSPubKey:    pk,
		BLSSecretKey: sk,
	}
}

// KeysJSON is a struct used to marshal / unmarshal fields to a encrypted file.
type KeysJSON struct {
	SecretKeyBLS []byte `json:"secret_key_bls"`
	PublicKeyBLS []byte `json:"public_key_bls"`
}

// NewFromFile reads keys from a file and tries to decrypt them.
func NewFromFile(password, path string) (*Keys, error) {
	keysJSONArr, err := fetchEncrypted(password, path)
	if err != nil {
		return nil, err
	}

	// transform []byte to keysJSON
	keysJSON := new(KeysJSON)

	err = json.Unmarshal(keysJSONArr, keysJSON)
	if err != nil {
		return nil, err
	}

	return &Keys{
		BLSPubKey:    keysJSON.PublicKeyBLS,
		BLSSecretKey: keysJSON.SecretKeyBLS,
	}, nil
}

// Save saves consensus keys to an encrypted file.
func (w *Keys) Save(password, path string) error {
	// Overwriting a consensus keys file may cause loss of secret keys
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadyExists
	}

	keysJSON := KeysJSON{
		SecretKeyBLS: w.BLSSecretKey,
		PublicKeyBLS: w.BLSPubKey,
	}

	// transform keysJSON to []byte
	data, err := json.Marshal(keysJSON)
	if err != nil {
		return err
	}

	// store it in a encrypted file
	if err := saveEncrypted(data, password, path); err != nil {
		return err
	}

	return nil
}

// saveEncrypted saves a []byte to a file.
func saveEncrypted(text []byte, password string, file string) error {
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

	return ioutil.WriteFile(file, gcm.Seal(nonce, nonce, text, nil), 0o600)
}

// fetchEncrypted load encrypted from file.
func fetchEncrypted(password string, file string) ([]byte, error) {
	digest := sha3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile(file) //nolint
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
	return gcm.Open(nil, nonce, ciphertext, nil)
}
