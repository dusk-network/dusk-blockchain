package rangeproof

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestProveBulletProof(t *testing.T) {

	var amount ristretto.Scalar
	amount.SetBigInt(big.NewInt(200))

	// Prove
	p, err := Prove(amount)
	fmt.Printf("%+v\n", p)
	assert.Equal(t, nil, err)
	_ = p

	// Verify
	// ok, err := Verify(p)
	// assert.Equal(t, nil, err)
	// assert.Equal(t, true, ok)
}

func TestComputeTau(t *testing.T) {
	a := ristretto.Scalar{}
	a.SetBigInt(big.NewInt(1))
	b := ristretto.Scalar{}
	b.SetBigInt(big.NewInt(2))
	c := ristretto.Scalar{}
	c.SetBigInt(big.NewInt(1))
	d := ristretto.Scalar{}
	d.SetBigInt(big.NewInt(1))
	e := ristretto.Scalar{}
	e.SetBigInt(big.NewInt(1))

	res := computeTaux(a, b, c, d, e)

	assert.Equal(t, int64(6), res.BigInt().Int64())

}

func TestComputeDelta(t *testing.T) {
	a := ristretto.Scalar{}
	a.SetBigInt(big.NewInt(1))
	b := ristretto.Scalar{}
	b.SetBigInt(big.NewInt(1))
	res, _ := computeDelta(a, b)
	t.Fail()
	c := ristretto.Scalar{}
	c.SetBigInt(big.NewInt(-15000))
	fmt.Println(res.BigInt().Int64())
	fmt.Println(c.BigInt().Int64())
	fmt.Println(c.BigInt().Int64() - res.BigInt().Int64())
}

/*
TODO: test values over the N threshold and named errors named errors
input: 2^N+1
output: error: value too large
*/
