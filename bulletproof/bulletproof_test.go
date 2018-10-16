package bulletproof

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestProveBulletProof(t *testing.T) {

	var amount ristretto.Scalar
	amount.SetBigInt(big.NewInt(200))
	// TODO: test values over the threshold, with named errors

	var gamma ristretto.Scalar
	gamma.Rand()

	_, err := Prove(amount, gamma)
	assert.Equal(t, nil, err)
}
