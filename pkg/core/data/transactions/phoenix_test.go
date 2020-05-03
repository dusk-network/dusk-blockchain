package transactions

import (
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	assert "github.com/stretchr/testify/require"
)

func TestRuskTxInputUnMarshal(t *testing.T) {
	assert := assert.New(t)
	ruskIn := mockTransactionInput()
	in := new(TransactionInput)
	assert.NoError(UTxIn(ruskIn, in))

	assert.Equal(ruskIn.Sk.A.Data, in.Sk.A.Data)
	assert.Equal(ruskIn.Nullifier.H.Data, in.Nullifier.H.Data)

	ruskOut := new(rusk.TransactionInput)
	assert.NoError(MTxIn(ruskOut, in))

	assert.Equal(in.Sk.A.Data, ruskOut.Sk.A.Data)
	assert.Equal(in.Nullifier.H.Data, ruskOut.Nullifier.H.Data)
}

func TestRuskTxOutUnMarshal(t *testing.T) {
	assert := assert.New(t)
	ruskIn := mockTransactionOutput()
	in := new(TransactionOutput)
	assert.NoError(UTxOut(ruskIn, in))

	assert.Equal(ruskIn.Pk.AG.Y, in.Pk.AG.Y)
	assert.Equal(ruskIn.BlindingFactor.Data, in.BlindingFactor.Data)

	ruskOut := new(rusk.TransactionOutput)
	assert.NoError(MTxOut(ruskOut, in))

	assert.Equal(in.Pk.AG.Y, ruskOut.Pk.AG.Y)
	assert.Equal(in.BlindingFactor.Data, ruskOut.BlindingFactor.Data)
}

func mockTransactionInput() *rusk.TransactionInput {
	return &rusk.TransactionInput{

		Note: mockNote1(),
		Sk: &rusk.SecretKey{
			A: &rusk.Scalar{Data: []byte{0x55, 0x66}},
			B: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		},
		Nullifier: &rusk.Nullifier{
			H: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		},
		MerkleRoot: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	}
}

func mockTransactionOutput() *rusk.TransactionOutput {
	return &rusk.TransactionOutput{
		Note: mockNote2(),
		Pk: &rusk.PublicKey{
			AG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
			BG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		},
		BlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	}
}
