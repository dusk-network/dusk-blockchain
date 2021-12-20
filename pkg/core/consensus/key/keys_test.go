// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package key

import (
	"bytes"
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
