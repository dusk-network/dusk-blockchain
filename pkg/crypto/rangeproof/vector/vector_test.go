package vector

import (
	"math/big"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
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

func TestSumPowers(t *testing.T) {

	var ten ristretto.Scalar
	ten.SetBigInt(big.NewInt(10))

	expectedValues := []int64{0, 1, 11, 111, 1111, 11111, 111111, 1111111, 11111111}

	for n, expected := range expectedValues {
		res := ScalarPowersSum(ten, uint64(n))

		assert.Equal(t, expected, res.BigInt().Int64())
	}
}

func TestSumPowersNEqualZeroOne(t *testing.T) {

	var one ristretto.Scalar
	one.SetOne()

	n := uint64(0)

	res := ScalarPowersSum(one, n)
	assert.Equal(t, int64(n), res.BigInt().Int64())

	n = 1
	res = ScalarPowersSum(one, n)
	assert.Equal(t, int64(n), res.BigInt().Int64())

}

func TestNeg(t *testing.T) {

	var one ristretto.Scalar
	one.SetBigInt(big.NewInt(1))
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	var three ristretto.Scalar
	three.SetBigInt(big.NewInt(3))

	a := []ristretto.Scalar{one, two, three}

	var minusOne ristretto.Scalar
	minusOne.SetBigInt(big.NewInt(-1))
	var minusTwo ristretto.Scalar
	minusTwo.SetBigInt(big.NewInt(-2))
	var minusThree ristretto.Scalar
	minusThree.SetBigInt(big.NewInt(-3))

	expect := []ristretto.Scalar{minusOne, minusTwo, minusThree}

	res := Neg(a)

	for i := range res {
		ok := res[i].Equals(&expect[i])
		assert.Equal(t, true, ok)
	}

}
func TestInnerProduct(t *testing.T) {
	var one ristretto.Scalar
	one.SetBigInt(big.NewInt(1))
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	var three ristretto.Scalar
	three.SetBigInt(big.NewInt(3))
	var four ristretto.Scalar
	four.SetBigInt(big.NewInt(4))
	var five ristretto.Scalar
	five.SetBigInt(big.NewInt(5))

	a := []ristretto.Scalar{one, two, three, four}
	b := []ristretto.Scalar{two, three, four, five}

	res, _ := InnerProduct(a, b)

	var expected ristretto.Scalar
	expected.SetBigInt(big.NewInt(40))

	ok := expected.Equals(&res)

	assert.Equal(t, true, ok)
}
