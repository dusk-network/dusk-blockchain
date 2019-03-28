package transactions_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	helper "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func TestEncodeDecodeBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx, err := helper.RandomBidTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a bid TX struct
	decTX := &transactions.Bid{}
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
	assert.Equal(transactions.BidType, decTX.TxType)
}

func TestMalformedBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx, err := helper.RandomBidTx(t, true)
	assert.Nil(tx)
	assert.NotNil(err)
}

func TestEqualsMethodBid(t *testing.T) {

	assert := assert.New(t)

	a, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}
