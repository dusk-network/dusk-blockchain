package checksum

import (
	"bytes"
	"errors"

	"golang.org/x/crypto/blake2b"
)

const Length = 4

func Generate(m []byte) []byte {
	digest := blake2b.Sum256(m)
	return digest[:Length]
}

func Extract(m []byte) ([]byte, []byte, error) {
	// First 4 bytes are the checksum
	if len(m) < Length {
		return nil, nil, errors.New("malformed checksum")
	}

	checksum := m[:Length]
	message := m[Length:]
	return message, checksum, nil
}

func Verify(m []byte, checksum []byte) bool {
	digest := blake2b.Sum256(m)
	return bytes.Equal(checksum, digest[:Length])
}
