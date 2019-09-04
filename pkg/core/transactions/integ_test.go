package transactions_test

import (
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestDecodeTransactions(t *testing.T) {
	txs := helper.RandomSliceOfTxs(t, 20)
	r := helper.TxsToReader(t, txs)

	decTxs, err := transactions.FromReader(r, uint64(len(txs)))
	assert.Nil(t, err)

	assert.Equal(t, len(txs), len(decTxs))

	for i := range txs {
		tx := txs[i]
		decTx := decTxs[i]
		assert.True(t, tx.Equals(decTx))
	}
}
