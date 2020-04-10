package helper

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/assert"
)

// RandomOutput returns a random output for testing
func RandomOutput(t *testing.T) *transactions.Output {
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
}

// RandomOutputs returns a slice of random outputs for testing
func RandomOutputs(t *testing.T, size int) transactions.Outputs {

	var outs transactions.Outputs

	for i := 0; i < size; i++ {
		out := RandomOutput(t)
		assert.NotNil(t, out)
		outs = append(outs, out)
	}

	return outs
}
