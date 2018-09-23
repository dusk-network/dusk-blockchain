package crypto

import (
	"crypto/rand"
	"errors"
	"strconv"

	"github.com/toghrulmaharramov/dusk-go/crypto/hash"
)

// RandEntropy takes an argument n and populates a byte slice of
// size n with random input.
func RandEntropy(n uint32) ([]byte, error) {

	b := make([]byte, n)
	a, err := rand.Read(b)

	if err != nil {
		return nil, errors.New("Error generating entropy " + err.Error())
	}
	if uint32(a) != n {
		return nil, errors.New("Error expected to read" + strconv.Itoa(int(n)) + " bytes instead read " + strconv.Itoa(a) + " bytes")
	}
	return b, nil
}

// Checksum hashes the data with Sha3256
// and returns the first four bytes
func Checksum(data []byte) ([]byte, error) {
	hash, err := hash.Sha3256(data)
	if err != nil {
		return nil, err
	}
	return hash[:4], err
}
