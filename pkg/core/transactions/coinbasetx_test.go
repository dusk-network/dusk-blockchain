package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeCoinbase(t *testing.T) {

	assert := assert.New(t)

	// random coinbase tx
	tx := randomCoinBaseTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a coinbase TX struct
	decTX := &Coinbase{}
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

func TestEqualsMethodCoinBase(t *testing.T) {

	assert := assert.New(t)

	a := randomCoinBaseTx(t, false)
	b := randomCoinBaseTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

func randomCoinBaseTx(t *testing.T, malformed bool) *Coinbase {

	proof := randomSlice(t, 2000)
	key := randomSlice(t, 32)
	address := randomSlice(t, 32)

	tx := NewCoinbase(proof, key, address)

	return tx
}
