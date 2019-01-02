package rangeproof

import (
	"errors"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
)

// BitCommitment will be a struct used to hold the values aL and aR
type BitCommitment struct {
	AL, AR []ristretto.Scalar
}

// BitCommit will take the value v producing aL and aR
// N.B. This has been specialised for N <= 64
func BitCommit(v *big.Int) BitCommitment {

	bc := BitCommitment{
		AL: make([]ristretto.Scalar, N),
		AR: make([]ristretto.Scalar, N),
	}

	var zero ristretto.Scalar
	zero.SetZero()
	var one ristretto.Scalar
	one.SetOne()
	var minusOne ristretto.Scalar
	minusOne.Neg(&one)

	num := v.Uint64()

	for i := 0; i < N; i++ {

		var rem uint64

		rem = num % 2
		num = num >> 1

		if rem == 0 {
			bc.AL[i] = zero
			bc.AR[i] = minusOne
		} else {
			bc.AL[i] = one
			bc.AR[i] = zero
		}

	}

	return bc
}

// Debug makes sure we have calculated
// the correct aR and aL values
func (b *BitCommitment) Debug(v *big.Int) error {

	var zero ristretto.Scalar
	zero.SetZero()
	var one ristretto.Scalar
	one.SetOne()

	testAL := big.NewInt(0)
	testAR := big.NewInt(0)

	for i := 0; i < N; i++ {

		var basePow, e = big.NewInt(2), big.NewInt(int64(i))
		basePow.Exp(basePow, e, nil)

		if b.AL[i].Equals(&one) {
			testAL = testAL.Add(testAL, basePow)
		}
		if b.AR[i].Equals(&zero) {
			testAR = testAR.Add(testAR, basePow)
		}
	}

	if testAL.Cmp(v) != 0 {
		return errors.New("[BitCommit(Debug)]: Wrong Value for AL")
	}

	if testAR.Cmp(v) != 0 {
		return errors.New("[BitCommit(Debug)]: Wrong Value for AL")
	}

	return nil
}
