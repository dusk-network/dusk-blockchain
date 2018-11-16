package rangeproof

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

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

	res, _ := innerProduct(a, b)

	var expected ristretto.Scalar
	expected.SetBigInt(big.NewInt(40))

	ok := expected.Equals(&res)

	assert.Equal(t, true, ok)
}
