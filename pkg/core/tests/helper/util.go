// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package helper

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	assert "github.com/stretchr/testify/require"
)

// TxsToBuffer converts a slice of transactions to a bytes.Buffer.
func TxsToBuffer(t *testing.T, txs []transactions.ContractCall) *bytes.Buffer {
	assert := assert.New(t)
	whole := new(bytes.Buffer)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		assert.NoError(transactions.Marshal(buf, tx))
		_, err := whole.ReadFrom(buf)
		assert.NoError(err)
	}

	return whole
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	if err != nil {
		panic(err)
	}
	return randSlice
}
