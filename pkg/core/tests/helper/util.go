package helper

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// TxsToReader converts a slice of transactions to an io.Reader
func TxsToReader(t *testing.T, txs []transactions.Transaction) io.Reader {
	buf := new(bytes.Buffer)

	for _, tx := range txs {
		err := tx.Encode(buf)
		if err != nil {
			assert.Nil(t, err)
		}
	}

	return bytes.NewReader(buf.Bytes())
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}
