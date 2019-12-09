package checksum_test

import (
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Benchmark which hashing function would perform better for checksum
// generation.

// SHA3-256
func BenchmarkSHA3256(b *testing.B) {
	message, _ := crypto.RandEntropy(200)
	for i := 0; i < b.N; i++ {
		_ = sha3.Sum256(message)
	}
}

// Blake2b
func BenchmarkBlake2b(b *testing.B) {
	message, _ := crypto.RandEntropy(200)
	for i := 0; i < b.N; i++ {
		_ = blake2b.Sum256(message)
	}
}
