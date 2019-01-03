package rangeproof

import (
	"math/big"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestDelta(t *testing.T) {
	var y, z ristretto.Scalar
	y.Rand()
	z.Rand()

	var zSq, zCu ristretto.Scalar
	zSq.SetZero()
	zCu.SetZero()

	zSq.Square(&z)
	zCu.Mul(&z, &zSq)

	var powerG ristretto.Scalar
	powerG.SetZero()

	var expY, exp2 ristretto.Scalar
	expY.SetOne()
	exp2.SetOne()

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	n := 256

	for i := 0; i < n; i++ {
		var a ristretto.Scalar
		a.Sub(&z, &zSq)  // z-z^2
		a.Mul(&a, &expY) // (z-z^2)y^i

		var b ristretto.Scalar
		b.Mul(&zCu, &exp2) // z^3 * 2^i

		var c ristretto.Scalar
		c.Sub(&a, &b) // (z-z^2)y^i - z^3 * 2^i

		powerG.Add(&powerG, &c)

		expY.Mul(&expY, &y)
		exp2.Mul(&exp2, &two)
	}

	have := computeDelta(y, z, uint32(n), 1)

	ok := powerG.Equals(&have)

	assert.Equal(t, true, ok)
}

// t0 = z^m * v[m] + D(y,z)
// N.B. There was a bug in computeT0; the below three test cases were used to debug this value, by fixing n,m,y,z, and v
// Removing them would be fine
func TestComputeT0(t *testing.T) {
	// This is the T0 calculated using the public data
	// case when n,m, z, v = 1, y = 0

	var n, m uint32
	n = 1
	m = 1

	var amount, y, z ristretto.Scalar
	y.SetZero()
	z.SetBigInt(big.NewInt(1))
	amount.SetOne()

	poly := polynomial{}
	t0 := poly.computeT0(y, z, []ristretto.Scalar{amount}, n, m)

	// compute Delta
	delta := computeDelta(y, z, n, m)

	var minusOne ristretto.Scalar
	minusOne.SetBigInt(big.NewInt(-1))

	assert.Equal(t, minusOne.Bytes(), delta.Bytes())

	var want ristretto.Scalar
	want.Add(&amount, &delta)

	ok := want.Equals(&t0)

	assert.Equal(t, true, ok)
}

func TestComputeT0Case2(t *testing.T) {
	// This is the T0 calculated using the public data
	// case when: n = 64, m = 2, y = 1, amount = 1, z = 1

	var n, m uint32
	n = 32
	m = 2

	var amount, y, z ristretto.Scalar
	y.SetOne()
	z.SetBigInt(big.NewInt(1))
	amount.SetOne()

	poly := polynomial{}
	t0 := poly.computeT0(y, z, []ristretto.Scalar{amount, amount}, n, m)

	// compute Delta
	delta := computeDelta(y, z, n, m) // (1-1)*sumY - (1 * 4294967295 *2)
	var deltaWant ristretto.Scalar
	deltaWant.SetBigInt(big.NewInt(-4294967295 * 2))
	assert.Equal(t, deltaWant.Bytes(), delta.Bytes())

	// z^m *v[m] = z^2 + z^3 = 1 + 1 = 2

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	var sum ristretto.Scalar
	sum.Add(&delta, &two)

	assert.Equal(t, t0.Bytes(), sum.Bytes())
}

func TestComputeT0Case3(t *testing.T) {
	// This is the T0 calculated using the public data
	// case when: n = 64, m = 2, y = 1, amount = 1, z = 2

	var n, m uint32
	n = 32
	m = 2

	var amount, y, z ristretto.Scalar
	y.SetOne()
	z.SetBigInt(big.NewInt(2))
	amount.SetOne()

	poly := polynomial{}
	t0 := poly.computeT0(y, z, []ristretto.Scalar{amount, amount}, n, m)

	// compute Delta
	delta := computeDelta(y, z, n, m) // (2 -4) * 64 - (8 * 4294967295 * 3)
	var deltaWant ristretto.Scalar
	d, ok := big.NewInt(0).SetString("-103079215208", 10)
	assert.Equal(t, true, ok)
	deltaWant.SetBigInt(d)
	assert.Equal(t, deltaWant.Bytes(), delta.Bytes())

	// z^m *v[m] = z^2 + z^3 = 4 + 8 = 2

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(12))

	var sum ristretto.Scalar
	sum.Add(&delta, &two)

	assert.Equal(t, t0.Bytes(), sum.Bytes())
}
