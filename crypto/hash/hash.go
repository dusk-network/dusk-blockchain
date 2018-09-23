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
