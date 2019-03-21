package transactions_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	helper "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func TestEncodeDecodeTLock(t *testing.T) {

	assert := assert.New(t)

	// random timelock tx
	tx := helper.RandomTLockTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a TimeLock TX struct
	decTX := &transactions.TimeLock{}
	err = decTX.Decode(buf)
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
	assert.Equal(transactions.TimelockType, decTX.TxType)
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
