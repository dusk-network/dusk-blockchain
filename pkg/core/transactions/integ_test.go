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

	decTxs := make([]transactions.Transaction, len(txs))
	for i := 0; i < len(txs); i++ {
		tx, err := transactions.Unmarshal(r)
		assert.Nil(t, err)
		decTxs[i] = tx
	}

	assert.Equal(t, len(txs), len(decTxs))

	for i := range txs {
		tx := txs[i]
		decTx := decTxs[i]
		assert.True(t, tx.Equals(decTx))
	}
}
