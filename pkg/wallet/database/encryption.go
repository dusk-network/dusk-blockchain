package database

import (
	"golang.org/x/crypto/sha3"
)

func createHash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}

func encrypt(data []byte, passphrase []byte) ([]byte, error) {
	return data, nil
}

func decrypt(data []byte, passphrase []byte) ([]byte, error) {
	return data, nil
}
