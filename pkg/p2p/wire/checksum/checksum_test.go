// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package checksum_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Benchmark which hashing function would perform better for checksum
// generation.

func BenchmarkSHA3256(b *testing.B) {
	message, _ := crypto.RandEntropy(200)
	for i := 0; i < b.N; i++ {
		_ = sha3.Sum256(message)
	}
}

func BenchmarkBlake2b(b *testing.B) {
	message, _ := crypto.RandEntropy(200)
	for i := 0; i < b.N; i++ {
		_ = blake2b.Sum256(message)
	}
}

func TestVerify(t *testing.T) {
	message, _ := crypto.RandEntropy(32)
	cs := checksum.Generate(message)
	assert.True(t, checksum.Verify(message, cs))
}

func TestInvalidLen(t *testing.T) {
	_, _, err := checksum.Extract([]byte{0, 0, 0})
	if err == nil {
		t.Errorf("invalid len err expected")
	}
}
