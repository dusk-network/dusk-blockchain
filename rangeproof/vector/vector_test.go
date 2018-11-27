package vector

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestVectorAdd(t *testing.T) {
	N := 64
	a := make([]ristretto.Scalar, N)
	b := make([]ristretto.Scalar, N)

	var one ristretto.Scalar
	one.SetOne()

	for i := range a {
		a[i] = one
	}
	for i := range b {
		b[i] = one
	}
	res, err := Add(a, b)
	assert.Equal(t, nil, err)

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	expected := make([]ristretto.Scalar, N)

	for i := range expected {
		expected[i] = two
	}

	assert.Equal(t, len(res), len(expected))
	for i := range res {
		ok := res[i].Equals(&expected[i])
		assert.Equal(t, true, ok)
	}
}
