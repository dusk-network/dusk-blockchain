package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeTLock(t *testing.T) {

	assert := assert.New(t)

	// random timelock tx
	tx := helper.RandomTLockTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := transactions.Marshal(buf, tx)
	assert.Nil(err)

	// Decode buffer into a TimeLock TX struct
	decTX, err := transactions.Unmarshal(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.TimelockType, decTX.(*transactions.TimeLock).TxType)
}

func TestEqualsMethodTimeLock(t *testing.T) {

	assert := assert.New(t)

	a := helper.RandomTLockTx(t, false)
	b := helper.RandomTLockTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}
