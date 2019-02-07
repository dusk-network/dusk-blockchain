package generation

import (
	"errors"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
)

// distfunc is the function used to generate the score using sigma distribution

// Score generates the score Q, for the block generation hase
func Score(d uint64, Y []byte) (uint64, error) {

	if len(Y) != 32 {
		return 0, errors.New("Y is not 256 bits (32 bytes) in length")
	}

	Yprime := Y[:len(Y)/2]

	YprimeBigInt := big.NewInt(0).SetBytes(Yprime)
	dBigInt := big.NewInt(0).SetUint64(d)

	var YPrimeRist, dRist, kRist ristretto.Scalar
	YPrimeRist.SetBigInt(YprimeBigInt)
	dRist.SetBigInt(dBigInt)
	kRist.Inverse(&YPrimeRist)

	//  f = d / k where k = 1/Y'
	var f ristretto.Scalar
	f.Mul(&dRist, &kRist)

	return f.BigInt().Uint64(), nil
}
