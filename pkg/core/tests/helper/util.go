package helper

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/assert"
)

// TxsToBuffer converts a slice of transactions to a bytes.Buffer.
func TxsToBuffer(t *testing.T, txs []transactions.ContractCall) *bytes.Buffer {
	whole := new(bytes.Buffer)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := transactions.Marshal(buf, tx)
		if err != nil {
			assert.Nil(t, err)
		}

		if _, err := whole.ReadFrom(buf); err != nil {
			assert.Nil(t, err)
		}
	}

	return whole
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}
