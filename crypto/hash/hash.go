package hash

import (
	"golang.org/x/crypto/sha3"
)

func Sha3256(data []byte) ([]byte, error) {
	hasher := sha3.New256()
	hasher.Reset()
	_, err := hasher.Write(data)
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func DoubleSha3256(data []byte) ([]byte, error) {
	h, err := Sha3256(data)
	if err != nil {
		return nil, err
	}
	hash, err := Sha3256(h)
	if err != nil {
		return nil, err
	}
	return hash, nil
}
