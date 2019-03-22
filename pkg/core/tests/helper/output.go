package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// RandomOutput returns a random output for testing
func RandomOutput(t *testing.T, malformed bool) (*transactions.Output, error) {

	var commSize, keySize, proofSize uint32 = 32, 32, 4500

	if malformed {
		commSize = 45 // This does not have an effect, while Bidding transaction can have clear text
		// and so the commitment size is not fixed
		keySize = 23
	}

	comm := RandomSlice(t, commSize)
	key := RandomSlice(t, keySize)
	proof := RandomSlice(t, proofSize)

	return transactions.NewOutput(comm, key, proof)
}

// RandomOutputs returns a slice of random outputs for testing
func RandomOutputs(t *testing.T, size int, malformed bool) transactions.Outputs {

	var outs transactions.Outputs

	for i := 0; i < size; i++ {
		out, err := RandomOutput(t, malformed)
		if !malformed {
			assert.Nil(t, err)
			assert.NotNil(t, out)
			outs = append(outs, out)
		}
	}

	return outs
}
