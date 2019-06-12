package key

import (
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// Checksum hashes the data with Sha3256
// and returns the first four bytes
func checksum(data []byte) (uint32, error) {
	hash, err := hash.Xxhash(data)
	if err != nil {
		return 0, err
	}
	checksum := binary.BigEndian.Uint32(hash[:4])
	return checksum, err
}

// CompareChecksum takes data and an expected checksum
// Returns true if the checksum of the given data is
// equal to the expected checksum
func compareChecksum(data []byte, want uint32) bool {
	got, err := checksum(data)
	if err != nil {
		return false
	}
	if got != want {
		return false
	}
	return true
}
