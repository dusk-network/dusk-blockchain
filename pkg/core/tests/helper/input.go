package helper

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
)

//const respAmount uint32 = 7

// RandomInput returns a random input for testing
func RandomInput(t *testing.T) *transactions.TransactionInput {
	/*
		amount := ristretto.Scalar{}
		amount.Rand()
		privKey := ristretto.Scalar{}
		privKey.Rand()
		mask := ristretto.Scalar{}
		mask.Rand()

		sigBuf := randomSignatureBuffer(t)

		in := transactions.NewInput(amount, privKey, mask)
		in.Signature = &mlsag.Signature{}
		if err := in.Signature.Decode(sigBuf, true); err != nil {
			t.Fatal(err)
		}

		in.KeyImage.Rand()
		return in
	*/
	return nil
}

// RandomInputs returns a slice of inputs of size `size` for testing
func RandomInputs(t *testing.T, size int) []*transactions.TransactionInput {
	/*
		var ins transactions.Inputs

		for i := 0; i < size; i++ {
			in := RandomInput(t)
			assert.NotNil(t, in)
			ins = append(ins, in)
		}

		return ins
	*/
	return nil
}
