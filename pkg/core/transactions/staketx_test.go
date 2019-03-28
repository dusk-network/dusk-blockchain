package transactions_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	helper "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func TestEncodeDecodeStake(t *testing.T) {

	assert := assert.New(t)

	// random Stake tx
	tx, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a Stake TX struct
	decTX := &transactions.Stake{}
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
	assert.Equal(transactions.StakeType, decTX.TxType)
}

func TestEqualsMethodStake(t *testing.T) {

	assert := assert.New(t)

	a, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}
