package rangeproof_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
)

func TestBitCommitToValue(t *testing.T) {

	expectedValues := []int64{200, 400, 5670, 1234, 4567, 890}

	for _, expected := range expectedValues {
		value := big.NewInt(expected)
		Commitment := rangeproof.BitCommit(value)

		val := big.NewInt(0)

		for i := len(Commitment.AL) - 1; i >= 0; i-- {
			var basePow, e = big.NewInt(2), big.NewInt(int64(i))
			basePow.Exp(basePow, e, nil)

			base10val := big.NewInt(0)
			base10val.Mul(basePow, Commitment.AL[i].BigInt())

			val.Add(val, base10val)
		}
		assert.Equal(t, expected, val.Int64())
	}

}

func TestEnsure(t *testing.T) {
	expectedValues := []int64{200, 400, 5670, 1234, 4567, 890}

	for _, expected := range expectedValues {

		value := big.NewInt(expected)
		Commitment := rangeproof.BitCommit(value)
		err := Commitment.Debug(value)

		assert.Equal(t, nil, err)

	}
}
