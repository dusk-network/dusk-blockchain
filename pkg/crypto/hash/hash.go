package hash

import (
	"golang.org/x/crypto/sha3"
)

// Sha3256 takes a byte slice
// and returns the SHA3-256 hash
func Sha3256(bs []byte) ([]byte, error) {

	h := sha3.New256()
	_, err := h.Write(bs)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), err

}

// Sha3512 takes a byte slice
// and returns the SHA3-512 hash
func Sha3512(bs []byte) ([]byte, error) {

	h := sha3.New512()
	_, err := h.Write(bs)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), err

}
