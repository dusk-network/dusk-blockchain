package hash

import (
	"hash"

	"golang.org/x/crypto/sha3"
)

// Sha3256 takes a byte slice
// and returns the SHA3-256 hash
func Sha3256(bs []byte) ([]byte, error) {
	return PerformHash(sha3.New256(), bs)
}

// PerformHash takes a generic hash.Hash and returns the hashed payload
func PerformHash(H hash.Hash, bs []byte) ([]byte, error) {
	_, err := H.Write(bs)
	if err != nil {
		return nil, err
	}
	return H.Sum(nil), err
}
