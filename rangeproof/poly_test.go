package rangeproof

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestSumPowersNEqualZeroOne(t *testing.T) {

	var one ristretto.Scalar
	one.SetOne()

	n := uint32(0)

	res := sumOfPowers(one, n)
	assert.Equal(t, int64(n), res.BigInt().Int64())

	n = 1
	res = sumOfPowers(one, n)
	assert.Equal(t, int64(n), res.BigInt().Int64())

}

// sumPowers and VectorPowerSum work the same way. Both are tested below
// XXX: Once computeDelta works, we will benchmark the faster variation and keep
func TestSumPowers(t *testing.T) {

	var ten ristretto.Scalar
	ten.SetBigInt(big.NewInt(10))

	expectedValues := []int64{0, 1, 11, 111, 1111, 11111, 111111, 1111111, 11111111}

	for n, expected := range expectedValues {
		res := sumOfPowers(ten, uint32(n))
		res2 := vecPowerSum(ten, uint64(n))
		assert.Equal(t, expected, res.BigInt().Int64())
		assert.Equal(t, res2.BigInt().Int64(), res.BigInt().Int64())
	}
}
