package transactions

import (
	"bytes"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// hashBytes loads all bytes into a buffer, then hashes it using sha3256
func hashBytes(encode func(io.Writer) error) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := encode(buf)
	if err != nil {
		return nil, err
	}
	return hash.Sha3256(buf.Bytes())
}
