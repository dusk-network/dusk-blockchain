package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx, err := randomBidTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a bid TX struct
	decTX := &Bid{}
	err = decTX.Decode(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.Hash()
	assert.Nil(err)

	decTxid, err := decTX.Hash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))
}

func TestMalformedBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx, err := randomBidTx(t, true)
	assert.Nil(tx)
	assert.NotNil(err)
}

func TestEqualsMethodBid(t *testing.T) {

	assert := assert.New(t)

	a, err := randomBidTx(t, false)
	assert.Nil(err)
	b, err := randomBidTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

func randomBidTx(t *testing.T, malformed bool) (*Bid, error) {

	var numInputs, numOutputs = 23, 34
	var lock uint64 = 20000
	var fee uint64 = 20
	var M = randomSlice(t, 32)

	if malformed {
		M = randomSlice(t, 12)
	}

	tx, err := NewBid(0, lock, fee, M)
	if err != nil {
		return tx, err
	}

	// Inputs
	tx.Inputs = randomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = randomOutputs(t, numOutputs, malformed)

	return tx, err
}
