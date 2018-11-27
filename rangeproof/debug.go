package rangeproof

import (
	"errors"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/vector"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/pedersen"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// Put all debug functions here

func debugProve(v, x, y, z ristretto.Scalar, l, r []ristretto.Scalar, aL, aR, sL, sR [N]ristretto.Scalar) error {
	ok := debugLxG(l, x, z, aL, aR, sL)
	if !ok {
		return errors.New("[Prove]: <l(x), G> is constructed incorrectly")
	}

	ok = debugRxHPrime(r, x, y, z, aR, sR)
	if !ok {
		return errors.New("[Prove]: <r(x), H'> is constructed incorrectly")
	}

	ok = debugsizeOfV(v.BigInt())
	if !ok {
		return errors.New("[Prove]: Value v is more than 2^N - 1")
	}

	return nil
}

// DEBUG

func debugT0(aL, aR [N]ristretto.Scalar, y, z ristretto.Scalar) ristretto.Scalar {

	aLMinusZ := vector.SubScalar(aL[:], z)

	aRPlusZ := vector.AddScalar(aR[:], z)

	yN := vector.ScalarPowers(y, N)

	hada, _ := vector.Hadamard(yN, aRPlusZ)

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vector.ScalarPowers(two, N)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	zsqMul2n := vector.MulScalar(twoN, zsq)

	rightIP, _ := vector.Add(zsqMul2n, hada)

	iP, _ := innerProduct(aLMinusZ, rightIP)

	return iP
}

// <l(x), G> =  <aL, G> + x<sL, G> +<-z1, G>
func debugLxG(l []ristretto.Scalar, x, z ristretto.Scalar, aL, aR, sL [N]ristretto.Scalar) bool {
	var P ristretto.Point
	P.SetZero()

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(65)

	G := ped.BaseVector.Bases[1:]

	lG, _ := vector.Exp(l, G, N, M)

	// <aL,G>
	aLG, _ := vector.Exp(aL[:], G, N, M)

	// x<sL, G>
	sLG, _ := vector.Exp(sL[:], G, N, M)
	var xsLG ristretto.Point
	xsLG.ScalarMult(&sLG, &x)
	// <-z1, G>
	var zNeg ristretto.Scalar
	zNeg.Neg(&z)
	zNegG, _ := vector.Exp(vector.FromScalar(zNeg, N), G, N, M)

	var rhs ristretto.Point
	rhs.SetZero()
	rhs.Add(&aLG, &xsLG)
	rhs.Add(&rhs, &zNegG)

	return lG.Equals(&rhs)
}

// < r(x), H'> = <aR, H> + x<sR, H> + <z*y^n + z^2 * 2^n , H'>
func debugRxHPrime(r []ristretto.Scalar, x, y, z ristretto.Scalar, aR, sR [N]ristretto.Scalar) bool {

	genData := []byte("dusk.BulletProof.vec1")

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(64)

	H := ped2.BaseVector.Bases

	Hprime := computeHprime(ped2.BaseVector.Bases, y)

	// <r(x), H'>
	rH, _ := vector.Exp(r, Hprime, N, M)

	// <aR,H>
	aRH, _ := vector.Exp(aR[:], H, N, M)

	// x<sR, H>
	sRH, _ := vector.Exp(sR[:], H, N, M)
	var xsRH ristretto.Point
	xsRH.ScalarMult(&sRH, &x)

	// y^n
	yN := vector.ScalarPowers(y, N)

	// 2^n
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vector.ScalarPowers(two, N)

	// z^2 * 2^n
	var zsq ristretto.Scalar
	zsq.Square(&z)
	zsqMul2n := vector.MulScalar(twoN, zsq)

	// z*y^n
	zMulYn := vector.MulScalar(yN, z)

	// p = z*y^n + z^2 * 2^n
	p, _ := vector.Add(zsqMul2n, zMulYn)

	// k = <p , H'>
	k, _ := vector.Exp(p, Hprime, N, M)

	var rhs ristretto.Point
	rhs.Add(&aRH, &xsRH)
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
