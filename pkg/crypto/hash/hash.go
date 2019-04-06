package hash

import (
	"hash"

	"github.com/OneOfOne/xxhash"
	"golang.org/x/crypto/sha3"
)

type Hash = hash.Hash

// Sha3256 takes a byte slice
// and returns the SHA3-256 hash
func Sha3256(bs []byte) ([]byte, error) {
	return PerformHash(sha3.New256(), bs)
}

// PerformHash takes a generic hash.Hash and returns the hashed payload
func PerformHash(H Hash, bs []byte) ([]byte, error) {
	_, err := H.Write(bs)
	if err != nil {
		return nil, err
	}
	return H.Sum(nil), err
}

// Sha3512 takes a byte slice
// and returns the SHA3-512 hash
func Sha3512(bs []byte) ([]byte, error) {
	return PerformHash(sha3.New512(), bs)
}

func Xxhash(bs []byte) ([]byte, error) {
	return PerformHash(xxhash.New64(), bs)
}
