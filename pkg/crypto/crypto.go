package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strconv"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
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
func Checksum(data []byte) (uint32, error) {
	hash, err := hash.Xxhash(data)
	if err != nil {
		return 0, err
	}
	checksum := binary.BigEndian.Uint32(hash[:4])
	return checksum, err
}

//CompareChecksum takes data and an expected checksum
// Returns true if the checksum of the given data is
// equal to the expected checksum
func CompareChecksum(data []byte, want uint32) bool {
	got, err := Checksum(data)
	if err != nil {
		return false
	}
	if got != want {
		return false
	}
	return true
}
