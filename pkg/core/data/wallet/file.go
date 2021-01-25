// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/sha3"
)

// saveEncrypted saves a []byte to a .dat file.
func saveEncrypted(text []byte, password string, file string) error {
	// Overwriting a seed file may cause loss of funds
	if _, err := os.Stat(file); err == nil {
		return ErrSeedFileExists
	}

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
