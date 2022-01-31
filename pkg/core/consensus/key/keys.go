// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package key

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"lukechampine.com/blake3"
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
	digest := blake3.Sum256([]byte(password))

	enc, err := AesCBCEncrypt(text, digest[:])
	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, enc, 0o600)
}

// fetchEncrypted load encrypted from file.
func fetchEncrypted(password string, file string) ([]byte, error) {
	digest := blake3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile(file) //nolint
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}

	blockSize := c.BlockSize()
	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]

	// CBC mode always works in whole blocks.
	if len(ciphertext)%blockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(c, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)
	// Unfill
	ciphertext, err = PKCS7UnPadding(ciphertext, blockSize)
	return ciphertext, err
}

func PKCS7UnPadding(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("pkcs7: Data is empty")
	}
	if length%blockSize != 0 {
		return nil, errors.New("pkcs7: Data is not block-aligned")
	}
	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen)
	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errors.New("pkcs7: Invalid padding")
	}
	return data[:length-padLen], nil
}

// pkcs7pad add pkcs7 padding
func PKCS7Padding(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 0 || blockSize > 256 {
		return nil, fmt.Errorf("pkcs7: Invalid block size %d", blockSize)
	} else {
		padLen := blockSize - len(data)%blockSize
		padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
		return append(data, padding...), nil
	}
}

//aes encryption, filling the 16 bits of the key key, 24, 32 respectively corresponding to AES-128, AES-192, or AES-256.
func AesCBCEncrypt(rawData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	//fill the original
	blockSize := block.BlockSize()
	rawData, err = PKCS7Padding(rawData, blockSize)
	if err != nil {
		panic(err)
	}
	
	// Initial vector IV must be unique, but does not need to be kept secret
	cipherText := make([]byte, blockSize+len(rawData))
	//block size 16
	iv := cipherText[:blockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	//block size and initial vector size must be the same
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[blockSize:], rawData)

	return cipherText, nil
}
