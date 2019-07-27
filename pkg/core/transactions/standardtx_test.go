package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeStandard(t *testing.T) {

	assert := assert.New(t)

	// random standard tx
	tx := helper.RandomStandardTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a standard TX struct
	decTX := &transactions.Standard{}
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
	assert.Equal(transactions.StandardType, decTX.TxType)
}

func TestEqualsMethodStandard(t *testing.T) {

	assert := assert.New(t)

	a := helper.RandomStandardTx(t, false)
	b := helper.RandomStandardTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}
