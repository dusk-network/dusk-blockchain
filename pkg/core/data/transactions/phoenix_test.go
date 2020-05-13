package transactions

import (
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	assert "github.com/stretchr/testify/require"
)

func TestRuskTxInputUnMarshal(t *testing.T) {
	assert := assert.New(t)
	ruskIn := RuskTransparentTxIn()
	in := new(TransactionInput)
	assert.NoError(UTxIn(ruskIn, in))

	assert.Equal(ruskIn.Nullifier.H.Data, in.Nullifier.H.Data)

	ruskOut := new(rusk.TransactionInput)
	assert.NoError(MTxIn(ruskOut, in))

	assert.Equal(in.Nullifier.H.Data, ruskOut.Nullifier.H.Data)
}

func TestRuskTxOutUnMarshal(t *testing.T) {
	assert := assert.New(t)
	ruskIn := RuskObfuscatedTxOut()
	in := new(TransactionOutput)
	assert.NoError(UTxOut(ruskIn, in))

	assert.Equal(ruskIn.Pk.AG.Y, in.Pk.AG.Y)
	assert.Equal(ruskIn.BlindingFactor.Data, in.BlindingFactor.Data)

	ruskOut := new(rusk.TransactionOutput)
	assert.NoError(MTxOut(ruskOut, in))

	assert.Equal(in.Pk.AG.Y, ruskOut.Pk.AG.Y)
	assert.Equal(in.BlindingFactor.Data, ruskOut.BlindingFactor.Data)
}
