package rangeproof

import (
	"errors"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// BitCommitment will be a struct used to hold the values aL and aR
type BitCommitment struct {
	AL, AR [N]ristretto.Scalar
}

// BitCommit will take the value v producing aL and aR
// TODO: clean up this function, looks a bit messy at the moment
// and not necessarily efficient
func BitCommit(v *big.Int) BitCommitment {

	bc := BitCommitment{}

	var zero ristretto.Scalar
	zero.SetZero()
	var one ristretto.Scalar
	one.SetOne()

	tempV := big.NewInt(v.Int64())

	for i := N - 1; i >= 0; i-- {
		var basePow, e = big.NewInt(2), big.NewInt(int64(i))
		basePow.Exp(basePow, e, nil)

		tV := big.NewInt(0) // we don't want to edit the tempV value, so we have this dummy val

		if (tV.Div(tempV, basePow)).Cmp(big.NewInt(0)) == 0 {
			bc.AL[i] = zero
		} else {

			bc.AL[i] = one
			tempV.Sub(tempV, basePow)
		}
		var aRi ristretto.Scalar
		aRi.Sub(&bc.AL[i], &one)
		bc.AR[i] = aRi
	}
	return bc
}

// Ensure makes sure we have calculated
// the correct aR and aL values
func (b *BitCommitment) Ensure(v *big.Int) (bool, error) {

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
		return false, errors.New("Wrong Value for AL")
	}

	if testAR.Cmp(v) != 0 {
		return false, errors.New("Wrong Value for AL")
	}

	return true, nil
}
