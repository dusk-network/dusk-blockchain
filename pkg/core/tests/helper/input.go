package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// RandomInput returns a random input for testing
func RandomInput(t *testing.T, malformed bool) (*transactions.Input, error) {

	var kiSize, txidSize, sigSize uint32 = 32, 32, 3400

	if malformed {
		kiSize = 45
		txidSize = 23
	}

	keyImage := RandomSlice(t, kiSize)
	pubkey := RandomSlice(t, txidSize)
	pseudoComm := RandomSlice(t, txidSize)
	sig := RandomSlice(t, sigSize)

	return transactions.NewInput(keyImage, pubkey, pseudoComm, sig)
}

// RandomInputs returns a slice of inputs of size `size` for testing
func RandomInputs(t *testing.T, size int, malformed bool) transactions.Inputs {

	var ins transactions.Inputs

	for i := 0; i < size; i++ {
		in, err := RandomInput(t, malformed)
		if !malformed {
			assert.Nil(t, err)
			assert.NotNil(t, in)
			ins = append(ins, in)
		}
	}

	return ins
}
