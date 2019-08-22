package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeStake(t *testing.T) {

	assert := assert.New(t)

	// random Stake tx
	tx, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = transactions.Marshal(buf, tx)
	assert.Nil(err)

	// Decode buffer into a Stake TX struct
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
	assert.Equal(transactions.StakeType, decTX.(*transactions.Stake).TxType)
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
