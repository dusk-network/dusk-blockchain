package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-crypto/hash"
)

// hashBytes loads all bytes into a buffer, then hashes it using sha3256
func hashBytes(buf *bytes.Buffer) ([]byte, error) {
	return hash.Sha3256(buf.Bytes())
}
