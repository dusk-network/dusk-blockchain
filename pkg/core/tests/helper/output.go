package helper

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
)

// RandomOutput returns a random output for testing
func RandomOutput(t *testing.T) *transactions.TransactionOutput {
	/*
		seed := RandomSlice(t, 128)
		keyPair := key.NewKeyPair(seed)

		r := ristretto.Scalar{}
		r.Rand()
		amount := ristretto.Scalar{}
		amount.Rand()
		encAmount := ristretto.Scalar{}
		encAmount.Rand()
		encMask := ristretto.Scalar{}
		encMask.Rand()

		output := transactions.NewOutput(r, amount, 0, *keyPair.PublicKey())

		output.EncryptedAmount = encAmount
		output.EncryptedMask = encMask
		return output
	*/
	return nil
}

// RandomOutputs returns a slice of random outputs for testing
func RandomOutputs(t *testing.T, size int) []*transactions.TransactionOutput {
	/*

		var outs transactions.Outputs

		for i := 0; i < size; i++ {
			out := RandomOutput(t)
			assert.NotNil(t, out)
			outs = append(outs, out)
		}

		return outs
	*/
	return nil
}
