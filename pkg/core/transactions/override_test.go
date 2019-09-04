package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

// Test that the tx type has overriden the standard hash function
func TestBidHashNotStandard(t *testing.T) {

	assert := assert.New(t)

	var txs []transactions.Transaction

	// Possible tye

	bidTx, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	txs = append(txs, bidTx)

	timeLockTx := helper.RandomTLockTx(t, false)
	txs = append(txs, timeLockTx)

	stakeTx, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	txs = append(txs, stakeTx)

	coinBaseTx := helper.RandomCoinBaseTx(t, false)
	assert.Nil(err)
	txs = append(txs, coinBaseTx)

	for _, tx := range txs {
		standardHash, txHash := calcTxAndStandardHash(t, tx)
		assert.False(bytes.Equal(standardHash, txHash))
	}

}

// calcTxAndStandardHash calculates the hash for the transaction and
// then the hash for the underlying standardTx. This ensures that the txhash being used,
// is not for the standardTx, unless this is explicitly called.
func calcTxAndStandardHash(t *testing.T, tx transactions.Transaction) ([]byte, []byte) {

	standard := tx.StandardTX()
	standardHash, err := standard.CalculateHash()

	txHash, err := tx.CalculateHash()
	assert.Nil(t, err)

	return standardHash, txHash
}
