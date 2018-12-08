package rangeproof

import (
	"errors"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/vector"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/pedersen"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// Put all debug functions here

func debugProve(x, y, z ristretto.Scalar, v, l, r []ristretto.Scalar, aL, aR, sL, sR []ristretto.Scalar) error {
	ok := debugLxG(l, x, z, aL, aR, sL)
	if !ok {
		return errors.New("[DEBUG]: <l(x), G> is constructed incorrectly")
	}

	ok = debugRxHPrime(r, x, y, z, aR, sR)
	if !ok {
		return errors.New("[DEBUG]: <r(x), H'> is constructed incorrectly")
	}

	for i := range v {

		ok = debugsizeOfV(v[i].BigInt())
		if !ok {
			return errors.New("[DEBUG]: Value v is more than 2^N - 1")
		}
	}

	return nil
}

// DEBUG

func debugT0(aL, aR []ristretto.Scalar, y, z ristretto.Scalar) ristretto.Scalar {

	aLMinusZ := vector.SubScalar(aL, z)

	aRPlusZ := vector.AddScalar(aR, z)

	yNM := vector.ScalarPowers(y, uint32(N*M))

	hada, _ := vector.Hadamard(yNM, aRPlusZ)

	zMTwoN := sumZMTwoN(z)

	rightIP, _ := vector.Add(zMTwoN, hada)

	iP, _ := innerProduct(aLMinusZ, rightIP)

	return iP
}

// <l(x), G> =  <aL, G> + x<sL, G> +<-z1, G>
func debugLxG(l []ristretto.Scalar, x, z ristretto.Scalar, aL, aR, sL []ristretto.Scalar) bool {
	var P ristretto.Point
	P.SetZero()

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32((N * M) + 1))

	G := ped.BaseVector.Bases[1:]

	lG, _ := vector.Exp(l, G, N, M)

	// <aL,G>
	aLG, err := vector.Exp(aL, G, N, M)
	if err != nil {
		return false
	}
	// x<sL, G>
	sLG, err := vector.Exp(sL, G, N, M)
	if err != nil {
		return false
	}
	var xsLG ristretto.Point
	xsLG.ScalarMult(&sLG, &x)
	// <-z1, G>
	var zNeg ristretto.Scalar
	zNeg.Neg(&z)
	zNegG, err := vector.Exp(vector.FromScalar(zNeg, uint32(N*M)), G, N, M)
	if err != nil {
		return false
	}
	var rhs ristretto.Point
	rhs.SetZero()
	rhs.Add(&aLG, &xsLG)
	rhs.Add(&rhs, &zNegG)

	return lG.Equals(&rhs)
}

// < r(x), H'> = <aR, H> + x<sR, H> + <z*y^(n*m), H'> + sum( (< <z^(j+1),2^n>, H') ) from j = 1 to j = m
func debugRxHPrime(r []ristretto.Scalar, x, y, z ristretto.Scalar, aR, sR []ristretto.Scalar) bool {

	genData := []byte("dusk.BulletProof.vec1")

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32((N * M)))

	H := ped2.BaseVector.Bases

	Hprime := computeHprime(ped2.BaseVector.Bases, y)

	// <r(x), H'>
	rH, err := vector.Exp(r, Hprime, N, M)
	if err != nil {
		return false
	}
	// <aR,H>
	aRH, err := vector.Exp(aR, H, N, M)
	if err != nil {
		return false
	}
	// x<sR, H>
	sRH, err := vector.Exp(sR, H, N, M)
	if err != nil {
		return false
	}
	var xsRH ristretto.Point
	xsRH.SetZero()
	xsRH.ScalarMult(&sRH, &x)

	// y^(n*m)
	yNM := vector.ScalarPowers(y, uint32(N*M))

	// z*y^nm
	zMulYn := vector.MulScalar(yNM, z)

	// p = <z*y^nm , H'>
	p, err := vector.Exp(zMulYn, Hprime, N, M)
	if err != nil {
		return false
	}
	// k = sum( (< <z^(j+1),2^n>, H') ) from j = 1 to j = m
	k, err := vector.Exp(sumZMTwoN(z), Hprime, N, M)
	if err != nil {
		return false
	}
	var rhs ristretto.Point
	rhs.Add(&aRH, &xsRH)
	rhs.Add(&rhs, &p)
	rhs.Add(&rhs, &k)

	return rH.Equals(&rhs)
}

// debugsizeOfV returns true if v is less than 2^N - 1
func debugsizeOfV(v *big.Int) bool {
	var twoN, e, one = big.NewInt(2), big.NewInt(int64(N)), big.NewInt(int64(1))
	twoN.Exp(twoN, e, nil)
	twoN.Sub(twoN, one)

	cmp := v.Cmp(twoN)
	return (cmp == -1)
}
