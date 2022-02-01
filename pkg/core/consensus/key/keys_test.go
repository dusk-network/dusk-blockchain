// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package key

import (
	"bytes"
	b64 "encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRandKeys(t *testing.T) {
	k := NewRandKeys()
	assert.NotNil(t, k.BLSPubKey)
	assert.NotNil(t, k.BLSSecretKey)

	k2 := NewRandKeys()
	assert.NotNil(t, k2.BLSPubKey)
	assert.NotNil(t, k2.BLSSecretKey)

	assert.True(t, !bytes.Equal(k.BLSPubKey, k2.BLSPubKey))
	assert.True(t, !bytes.Equal(k.BLSSecretKey, k2.BLSSecretKey))
}

func TestEncryptedKeysFromFile(t *testing.T) {
	const (
		password = "greatpassword"
	)

	walletenc := "s9CrfaDJY9H5wsBnj9qvRpXaocXHCxJGRd/hhhVfpvnIsW4SuV5AAeS+tVaW6NIVnjVShk6fthtC" +
		"iAhslbb50lzNxE0pAaiWxuAW4skZxoQCbxXoAK7SMcRez6A4cY94uKpmoakFCji1u8weg7qVzV8C" +
		"OXxpPlDo62UFwiuH+ZNhnktq+wD3VNAd/raBDQed+59nLoeTD1gE7mEgmxSncUd8+eDHzhmAMdUK" +
		"5hRU6u/LuQO/vZjQO6WlognYKhLzmr1kSvNTk/hctjmtLOW08LKlwigSRLpaJV1VVZP81HFnVZRS" +
		"VfRcVAJT43m+Vwpe"

	sDec, _ := b64.StdEncoding.DecodeString(walletenc)
	path := os.TempDir() + "/consensus-test-fromrusk.keys"
	err := os.WriteFile(path, sDec, 0644)
	assert.Nil(t, err)

	// try to load consensus keys with wrong pass
	k2, err := NewFromFile(password, path)
	assert.NotNil(t, k2)
	assert.Nil(t, err)
}

func TestEncryptedKeys(t *testing.T) {
	const (
		password = "password"
	)

	k := NewRandKeys()
	assert.NotNil(t, k.BLSPubKey)
	assert.NotNil(t, k.BLSSecretKey)

	path := os.TempDir() + "/consensus-test-1.keys"
	os.Remove(path)

	err := k.Save(password, path)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	// try to load consensus keys with wrong pass
	k2, err := NewFromFile("wrong_pass", path)
	assert.NotNil(t, err)
	assert.Nil(t, k2)

	// try to load consensus keys with wrong pass
	k3, err := NewFromFile(password, path)
	assert.Nil(t, err)

	assert.True(t, bytes.Equal(k3.BLSPubKey, k.BLSPubKey))
	assert.True(t, bytes.Equal(k3.BLSSecretKey, k.BLSSecretKey))
}

func TestAlreadyExists(t *testing.T) {
	const (
		password = "password"
	)

	k := NewRandKeys()
	assert.NotNil(t, k.BLSPubKey)
	assert.NotNil(t, k.BLSSecretKey)

	path := os.TempDir() + "/consensus-test-2.keys"
	os.Remove(path)

	err := k.Save(password, path)
	assert.Nil(t, err)

	err = k.Save(password, path)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}
