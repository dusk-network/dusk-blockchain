package rangeproof

import (
	"math/big"

	"github.com/pkg/errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/vector"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/pedersen"
)

// Put all debug functions here
func debugProve(x, y, z ristretto.Scalar, v, l, r []ristretto.Scalar, aL, aR, sL, sR []ristretto.Scalar) error {

	ok, err := debugLxG(l, x, z, aL, aR, sL)
	if !ok {
		return errors.Wrap(err, "[DEBUG]: <l(x), G> is constructed incorrectly")
	}

	ok, err = debugRxHPrime(r, x, y, z, aR, sR)
	if !ok {
		return errors.Wrap(err, "[DEBUG]: <r(x), H'> is constructed incorrectly")
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

func debugT0(aL, aR []ristretto.Scalar, y, z ristretto.Scalar) (ristretto.Scalar, error) {

	aLMinusZ := vector.SubScalar(aL, z)

	aRPlusZ := vector.AddScalar(aR, z)

	yNM := vector.ScalarPowers(y, uint32(N*M))

	hada, err := vector.Hadamard(yNM, aRPlusZ)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	zMTwoN := sumZMTwoN(z)

	rightIP, err := vector.Add(zMTwoN, hada)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	iP, err := vector.InnerProduct(aLMinusZ, rightIP)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	return iP, nil
}

// <l(x), G> =  <aL, G> + x<sL, G> +<-z1, G>
func debugLxG(l []ristretto.Scalar, x, z ristretto.Scalar, aL, aR, sL []ristretto.Scalar) (bool, error) {

	var P ristretto.Point
	P.SetZero()

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32((N * M)))

	G := ped.BaseVector.Bases

	lG, err := vector.Exp(l, G, N, M)
	if err != nil {
		return false, errors.Wrap(err, "<l(x), G>")
	}
	// <aL,G>
	aLG, err := vector.Exp(aL, G, N, M)
	if err != nil {
		return false, errors.Wrap(err, "<aL,G>")
	}
	// x<sL, G>
	sLG, err := vector.Exp(sL, G, N, M)
	if err != nil {
		return false, errors.Wrap(err, "x<sL, G>")
	}
	var xsLG ristretto.Point
	xsLG.ScalarMult(&sLG, &x)

	// <-z1, G>
	var zNeg ristretto.Scalar
	zNeg.Neg(&z)
	zNegG, err := vector.Exp(vector.FromScalar(zNeg, uint32(N*M)), G, N, M)
	if err != nil {
		return false, errors.Wrap(err, "<-z1, G>")
	}
	var rhs ristretto.Point
	rhs.SetZero()
	rhs.Add(&aLG, &xsLG)
	rhs.Add(&rhs, &zNegG)

	return lG.Equals(&rhs), nil
}

// < r(x), H'> = <aR, H> + x<sR, H> + <z*y^(n*m), H'> + sum( (< <z^(j+1),2^n>, H') ) from j = 1 to j = m
func debugRxHPrime(r []ristretto.Scalar, x, y, z ristretto.Scalar, aR, sR []ristretto.Scalar) (bool, error) {

	genData := []byte("dusk.BulletProof.vec1")

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32((N * M)))

	H := ped2.BaseVector.Bases

	Hprime := computeHprime(H, y)

	// <r(x), H'>
	rH, err := vector.Exp(r, Hprime, N, M)
	if err != nil {
		return false, errors.Wrap(err, "<r(x), H'>")
	}

	// <aR,H>
	aRH, err := vector.Exp(aR, H, N, M)
	if err != nil {
		return false, errors.Wrap(err, "<aR,H>")
	}
	// x<sR, H>
	sRH, err := vector.Exp(sR, H, N, M)
	if err != nil {

		return false, errors.Wrap(err, "x<sR, H>")
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
		return false, errors.Wrap(err, "<z*y^nm , H'>")
	}
	// k = sum( (< <z^(j+1) * 2^n>, H') ) from j = 1 to j = m
	k, err := vector.Exp(sumZMTwoN(z), Hprime, N, M)
	if err != nil {
		return false, errors.Wrap(err, "k = sum()...")
	}

	var rhs ristretto.Point
	rhs.SetZero()

	rhs.Add(&aRH, &xsRH)
	rhs.Add(&rhs, &p)
	rhs.Add(&rhs, &k)

	return rH.Equals(&rhs), nil
}

// debugsizeOfV returns true if v is less than 2^N - 1
func debugsizeOfV(v *big.Int) bool {
	var twoN, e, one = big.NewInt(2), big.NewInt(int64(N)), big.NewInt(int64(1))
	twoN.Exp(twoN, e, nil)
	twoN.Sub(twoN, one)

	cmp := v.Cmp(twoN)
	return (cmp == -1)
}
