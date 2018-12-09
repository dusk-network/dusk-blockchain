package rangeproof

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestProveBulletProof(t *testing.T) {

	m := 2 // XXX: for now needs to be a multiple of two

	amounts := []ristretto.Scalar{}

	for i := 0; i < m; i++ {

		var amount ristretto.Scalar

		n := rand.Int63()
		amount.SetBigInt(big.NewInt(n))

		amounts = append(amounts, amount)
	}

	// t.Fail()
	// Prove
	p, err := Prove(amounts, true)
	assert.Equal(t, nil, err)

	// Verify
	ok, err := Verify(p)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, ok)

}

func TestComputeMu(t *testing.T) {
	var one ristretto.Scalar
	one.SetOne()

	var expected ristretto.Scalar
	expected.SetBigInt(big.NewInt(2))

	res := computeMu(one, one, one)

	ok := expected.Equals(&res)

	assert.Equal(t, true, ok)
}

func BenchmarkProve(b *testing.B) {

	var amount ristretto.Scalar

	amount.SetBigInt(big.NewInt(100000))

	for i := 0; i < 100; i++ {

		// Prove
		Prove([]ristretto.Scalar{amount}, false)
	}

}
func BenchmarkVerify(b *testing.B) {

	var amount ristretto.Scalar

	amount.SetBigInt(big.NewInt(100000))
	p, _ := Prove([]ristretto.Scalar{amount}, false)

	for i := 0; i < 100; i++ {

		// Verify
		Verify(p)
	}

}
