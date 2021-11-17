// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package block

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/stretchr/testify/assert"
)

func TestTxFromBlock(t *testing.T) {
	assert := assert.New(t)

	// random block
	blk := &Block{
		Txs: transactions.RandContractCalls(3, 0, true),
	}

	// search for non-existing tx
	tx, err := blk.Tx([]byte{0, 0, 0, 0})
	assert.Error(err)
	assert.Nil(tx)

	// search for all existing tx
	for i := 0; i < len(blk.Txs); i++ {
		txid, _ := blk.Txs[i].CalculateHash()

		tx, err = blk.Tx(txid)
		assert.NoError(err)
		assert.NotNil(tx)
	}
}
