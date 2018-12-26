package rangeproof

import (
	"fmt"
	"math/big"

	"github.com/pkg/errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/fiatshamir"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/innerproduct"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/pedersen"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/vector"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/ristretto"
)

// N is number of bits in range
// So amount will be between 0..2^(N-1)
const N = 64

// M is the number of outputs
// for one bulletproof
var M = 1

// M is the maximum number of outputs allowed
// per bulletproof
const maxM = 32

// Proof is the constructed BulletProof
type Proof struct {
	V  []pedersen.Commitment // Curve points 32 bytes
	A  ristretto.Point       // Curve point 32 bytes
	S  ristretto.Point       // Curve point 32 bytes
	T1 ristretto.Point       // Curve point 32 bytes
	T2 ristretto.Point       // Curve point 32 bytes

	taux ristretto.Scalar //scalar
	mu   ristretto.Scalar //scalar
	t    ristretto.Scalar
	// XXX: Remove once debugging is completed
	l       []ristretto.Scalar // []scalar
	r       []ristretto.Scalar // []scalar
	IPProof *innerproduct.Proof
}

// Prove will take a scalar as a parameter and
// using zero knowledge, prove that it is [0, 2^N)
func Prove(v []ristretto.Scalar, debug bool) (Proof, error) {

	if len(v) < 1 {
		return Proof{}, errors.New("length of slice v is zero")
	}

	M = len(v)

	// commitment to values v
	Vs := make([]pedersen.Commitment, 0, M)
	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32((N * M)))

	// Hash for Fiat-Shamir
	hs := fiatshamir.HashCacher{[]byte{}}

	for _, amount := range v {
		// compute commmitment to v
		V := ped.CommitToScalar(amount)

		Vs = append(Vs, V)

		// update Fiat-Shamir
		hs.Append(V.Value.Bytes())
	}

	aLs := make([]ristretto.Scalar, 0, N*M)
	aRs := make([]ristretto.Scalar, 0, N*M)

	for i := range v {
		// Compute Bitcommits aL and aR to v
		BC := BitCommit(v[i].BigInt())
		aLs = append(aLs, BC.AL...)
		aRs = append(aRs, BC.AR...)
	}

	// Compute A
	A := computeA(ped, aLs, aRs)

	// // Compute S
	S, sL, sR := computeS(ped)

	// // update Fiat-Shamir
	hs.Append(A.Value.Bytes(), S.Value.Bytes())

	// compute y and z
	y, z := computeYAndZ(hs)

	// compute polynomial
	poly, err := computePoly(aLs, aRs, sL, sR, y, z)
	if err != nil {
		return Proof{}, errors.Wrap(err, "[Prove] - poly")
	}

	// Compute T1 and T2
	T1 := ped.CommitToScalar(poly.t1)
	T2 := ped.CommitToScalar(poly.t2)

	// update Fiat-Shamir
	hs.Append(z.Bytes(), T1.Value.Bytes(), T2.Value.Bytes())

	// compute x
	x := computeX(hs)
	// compute taux which is just the polynomial for the blinding factors at a point x
	taux := computeTaux(x, z, T1.BlindingFactor, T2.BlindingFactor, Vs)
	// compute mu
	mu := computeMu(x, A.BlindingFactor, S.BlindingFactor)

	// compute l dot r
	l, err := poly.computeL(x)
	if err != nil {
		return Proof{}, errors.Wrap(err, "[Prove] - l")
	}
	r, err := poly.computeR(x)
	if err != nil {
		return Proof{}, errors.Wrap(err, "[Prove] - r")
	}
	t, err := vector.InnerProduct(l, r)
	if err != nil {
		return Proof{}, errors.Wrap(err, "[Prove] - t")
	}

	// START DEBUG
	if debug {
		err := debugProve(x, y, z, v, l, r, aLs, aRs, sL, sR)
		if err != nil {
			return Proof{}, errors.Wrap(err, "[Prove] - debugProve")
		}

		// DEBUG T0
		testT0, err := debugT0(aLs, aRs, y, z)
		if err != nil {
			return Proof{}, errors.Wrap(err, "[Prove] - testT0")

		}
		if !testT0.Equals(&poly.t0) {
			return Proof{}, errors.New("[Prove]: Test t0 value does not match the value calculated from the polynomial")
		}

		polyt0 := poly.computeT0(y, z, v, N, uint32(M))
		if !polyt0.Equals(&poly.t0) {
			return Proof{}, errors.New("[Prove]: t0 value from delta function, does not match the polynomial t0 value(Correct)")
		}

		tPoly := poly.eval(x)
		if !t.Equals(&tPoly) {
			return Proof{}, errors.New("[Prove]: The t value computed from the t-poly, does not match the t value computed from the inner product of l and r")
		}
	}
	// End DEBUG

	// check if any challenge scalars are zero
	if x.IsNonZeroI() == 0 || y.IsNonZeroI() == 0 || z.IsNonZeroI() == 0 {
		return Proof{}, errors.New("[Prove] - One of the challenge scalars, x, y, or z was equal to zero. Generate proof again")
	}

	hs.Append(x.Bytes(), taux.Bytes(), mu.Bytes(), t.Bytes())

	// calculate inner product proof
	Q := ristretto.Point{}
	w := hs.Derive()
	Q.ScalarMult(&ped.BasePoint, &w)

	var yinv ristretto.Scalar
	yinv.Inverse(&y)
	Hpf := vector.ScalarPowers(yinv, uint32(N*M))

	genData = append(genData, uint8(1))
	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32(N * M))

	H := ped2.BaseVector.Bases
	G := ped.BaseVector.Bases

	ip, err := innerproduct.Generate(G, H, l, r, Hpf, Q)
	if err != nil {
		return Proof{}, errors.Wrap(err, "[Prove] -  ipproof")
	}

	return Proof{
		V:       Vs,
		A:       A.Value,
		S:       S.Value,
		T1:      T1.Value,
		T2:      T2.Value,
		l:       l,
		r:       r,
		t:       t,
		taux:    taux,
		mu:      mu,
		IPProof: ip,
	}, nil
}

// A = kH + aL*G + aR*H
func computeA(ped *pedersen.Pedersen, aLs, aRs []ristretto.Scalar) pedersen.Commitment {

	cA := ped.CommitToVectors(aLs, aRs)

	return cA
}

// S = kH + sL*G + sR * H
func computeS(ped *pedersen.Pedersen) (pedersen.Commitment, []ristretto.Scalar, []ristretto.Scalar) {

	sL, sR := make([]ristretto.Scalar, N*M), make([]ristretto.Scalar, N*M)
	for i := 0; i < N*M; i++ {
		var randA ristretto.Scalar
		randA.Rand()
		sL[i] = randA

		var randB ristretto.Scalar
		randB.Rand()
		sR[i] = randB
	}

	cS := ped.CommitToVectors(sL, sR)

	return cS, sL, sR
}

func computeYAndZ(hs fiatshamir.HashCacher) (ristretto.Scalar, ristretto.Scalar) {

	var y ristretto.Scalar
	y.Derive(hs.Result())

	var z ristretto.Scalar
	z.Derive(y.Bytes())

	return y, z
}
func computeX(hs fiatshamir.HashCacher) ristretto.Scalar {
	var x ristretto.Scalar
	x.Derive(hs.Result())
	return x
}

// compute polynomial for blinding factors l61
// N.B. tau1 means tau superscript 1
// taux = t1Blind * x + t2Blind * x^2 + (sum(z^n+1 * vBlind[n-1])) from n = 1 to n = m
func computeTaux(x, z, t1Blind, t2Blind ristretto.Scalar, vBlinds []pedersen.Commitment) ristretto.Scalar {
	tau1X := t1Blind.Mul(&x, &t1Blind)

	var xsq ristretto.Scalar
	xsq.Square(&x)

	tau2Xsq := t2Blind.Mul(&xsq, &t2Blind)

	var zN ristretto.Scalar
	zN.Square(&z) // start at zSq

	var zNBlindSum ristretto.Scalar
	zNBlindSum.SetZero()

	for i := range vBlinds {
		zNBlindSum.MulAdd(&zN, &vBlinds[i].BlindingFactor, &zNBlindSum)
		zN.Mul(&zN, &z)
	}

	var res ristretto.Scalar
	res.Add(tau1X, tau2Xsq)
	res.Add(&res, &zNBlindSum)

	return res
}

// alpha is the blinding factor for A
// rho is the blinding factor for S
// mu = alpha + rho * x
func computeMu(x, alpha, rho ristretto.Scalar) ristretto.Scalar {

	var mu ristretto.Scalar

	mu.MulAdd(&rho, &x, &alpha)

	return mu
}

// P = -mu*BlindingFactor + A + xS + Si(y,z)
func computeP(Blind, A, S ristretto.Point, mu, x, y, z ristretto.Scalar) (ristretto.Point, error) {

	var P ristretto.Point
	P.SetZero()

	var xS ristretto.Point
	xS.ScalarMult(&S, &x)

	Si, err := computeSi(y, z)
	if err != nil {
		return ristretto.Point{}, errors.Wrap(err, "[computeP]")
	}
	// Add -mu * blindingFactor
	var muB ristretto.Point
	muB.ScalarMult(&Blind, &mu)
	muB.Neg(&muB)

	P.Add(&A, &xS)
	P.Add(&P, &muB)
	P.Add(&P, &Si)

	return P, nil
}

// Si = -z<1,G> + z<1,H> + <(y^-nm Had z^2 *(z^0 * 2^n ... z^m-1 * 2^n)), H>
func computeSi(y, z ristretto.Scalar) (ristretto.Point, error) {

	var yinv ristretto.Scalar
	yinv.Inverse(&y)

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32((N * M)))

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32(N * M))

	H := ped2.BaseVector.Bases
	G := ped.BaseVector.Bases

	yinvNM := vector.ScalarPowers(yinv, uint32(N*M))

	concatZM2N := sumZMTwoN(z)

	concatZM2N, err := vector.Hadamard(yinvNM, concatZM2N)
	if err != nil {
		return ristretto.Point{}, errors.Wrap(err, "[ComputeSI] - concatZM2N")
	}
	a, err := vector.Exp(concatZM2N, H, N, M)
	if err != nil {
		return ristretto.Point{}, errors.Wrap(err, "[ComputeSI] - a")
	}

	// b =  z<1,H> - z<1, G> = z( <1,H> - <1,G>)
	var b ristretto.Point
	b.SetZero()

	for i := range H {
		b.Add(&b, &H[i])
		b.Sub(&b, &G[i])
	}
	b.ScalarMult(&b, &z)

	var Si ristretto.Point
	Si.SetZero()

	Si.Add(&a, &b)

	return Si, nil
}

// computeHprime will take a a slice of points H, with a scalar y
// and return a slice of points Hprime,  such that Hprime = y^-n * H
func computeHprime(H []ristretto.Point, y ristretto.Scalar) []ristretto.Point {
	Hprimes := make([]ristretto.Point, len(H))

	var yInv ristretto.Scalar
	yInv.Inverse(&y)

	invYInt := yInv.BigInt()

	for i, p := range H {
		// compute y^-i
		var invYPowInt big.Int
		invYPowInt.Exp(invYInt, big.NewInt(int64(i)), nil)

		var invY ristretto.Scalar
		invY.SetBigInt(&invYPowInt)

		var hprime ristretto.Point
		hprime.ScalarMult(&p, &invY)

		Hprimes[i] = hprime
	}

	return Hprimes
}

// P = lG + rH
func computeLGRH(y, mu ristretto.Scalar, l, r []ristretto.Scalar) (ristretto.Point, error) {

	var P ristretto.Point
	P.SetZero()

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32(N * M))

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32(N * M))

	H := ped2.BaseVector.Bases
	Hprime := computeHprime(H, y)
	G := ped.BaseVector.Bases

	rH, err := vector.Exp(r, Hprime, N, M)
	if err != nil {
		return ristretto.Point{}, errors.Wrap(err, "[computeLGRH] - rH")
	}
	lG, err := vector.Exp(l, G, N, M)
	if err != nil {
		return ristretto.Point{}, errors.Wrap(err, "[computeLGRH] - lG")
	}

	P.Add(&lG, &rH)

	return P, nil
}

// Verify takes a bullet proof and
// returns true only if the proof was valid
func Verify(p Proof) (bool, error) {

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32(N * M))

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32(N * M))

	G := ped.BaseVector.Bases
	H := ped2.BaseVector.Bases

	if len(p.l) != len(p.r) {
		return false, errors.New("[Verify]: Sizes of l and r do not match")
	}
	if len(p.l) <= 0 {
		return false, errors.New("[Verify]: size of l or r cannot be zero; empty proof")
	}

	// Reconstruct the challenges
	hs := fiatshamir.HashCacher{[]byte{}}
	for _, V := range p.V {
		hs.Append(V.Value.Bytes())
	}

	hs.Append(p.A.Bytes(), p.S.Bytes())
	y, z := computeYAndZ(hs)
	hs.Append(z.Bytes(), p.T1.Bytes(), p.T2.Bytes())
	x := computeX(hs)
	hs.Append(x.Bytes(), p.taux.Bytes(), p.mu.Bytes(), p.t.Bytes())
	w := hs.Derive()

	// var c ristretto.Scalar
	// c.Rand()

	// prove l(x) and r(x) is correct <lG, rH> = A+ xS + z<1,H> - z<1,G> + sum...
	Pleft, err := computeLGRH(y, p.mu, p.l, p.r)
	if err != nil {
		return false, errors.Wrap(err, "[Verify] - Pleft")
	}
	PRight, err := computeP(ped.BlindPoint, p.A, p.S, p.mu, x, y, z)
	if err != nil {
		return false, errors.Wrap(err, "[Verify] - PRight")
	}
	if !Pleft.Equals(&PRight) {
		return false, errors.New("[Verify]: Proof for l(x),r(x) is wrong")
	}

	ok := verifyT0(x, y, z, p.taux, p.t, ped.BasePoint, ped.BlindPoint, p.T1, p.T2, p.V)
	if !ok {
		fmt.Println("a")
		return false, errors.New("Check1 failed")
	}

	// PRight, err = computeP(ped.BlindPoint, p.A, p.S, p.mu, x, y, z)
	// if err != nil {
	// 	fmt.Println("b")
	// 	return false, errors.Wrap(err, "[Verify] - PRight")
	// }

	ok, err = verifyIP(PRight, p.IPProof, p.mu, x, y, z, p.t, w, p.A, ped.BasePoint, ped.BlindPoint, p.S, G, H)
	if err != nil {
		fmt.Println("c")
		return false, err
	}
	if !ok {
		fmt.Println("d")
		return false, errors.Wrap(err, "<lx, rx> check Failed")
	}

	return true, nil
}

func verifyT0(x, y, z, taux, t ristretto.Scalar, G, H, T1, T2 ristretto.Point, V []pedersen.Commitment) bool {

	delta := computeDelta(y, z, N, uint32(M))

	var c1, c2, c3, c4 ristretto.Point

	// Scalars
	var tMinusDelta ristretto.Scalar
	tMinusDelta.Sub(&t, &delta)

	var xSq ristretto.Scalar
	xSq.Square(&x)

	// summing up individual components
	c1.ScalarMult(&G, &tMinusDelta)

	c2.ScalarMult(&H, &taux)

	c3.ScalarMult(&T1, &x)

	c4.ScalarMult(&T2, &xSq)

	var zM ristretto.Scalar
	zM = z

	var zMV ristretto.Point
	var c5 ristretto.Point

	for i := 0; i < M; i++ {
		zM.Mul(&zM, &z)
		zMV.ScalarMult(&V[i].Value, &zM)
		c5.Add(&c5, &zMV)
	}

	var sum ristretto.Point
	sum.SetZero()
	sum.Add(&c1, &c2)
	sum.Add(&sum, &c3)
	sum.Add(&sum, &c4)
	sum.Add(&sum, &c5)

	var zero ristretto.Point
	zero.SetZero()

	return sum.Equals(&zero)
}

func verifyIP(P ristretto.Point, ipproof *innerproduct.Proof, mu, x, y, z, t, w ristretto.Scalar, A, G, H, S ristretto.Point, GVec, HVec []ristretto.Point) (bool, error) {

	uSq, uInvSq, s := ipproof.VerifScalars()

	sInv := make([]ristretto.Scalar, len(s))
	copy(sInv, s)

	// reverse s
	for i, j := 0, len(sInv)-1; i < j; i, j = i+1, j-1 {
		sInv[i], sInv[j] = sInv[j], sInv[i]
	}

	var tw ristretto.Scalar
	tw.Mul(&t, &w)

	var c1 ristretto.Point
	c1.ScalarMult(&G, &tw)

	as := vector.MulScalar(s, ipproof.A)

	c2, err := vector.Exp(as, GVec, len(GVec), 1)
	if err != nil {
		return false, err
	}

	var yinv ristretto.Scalar
	yinv.Inverse(&y)
	Hpf := vector.ScalarPowers(yinv, uint32(N*M))

	bDivS := vector.MulScalar(sInv, ipproof.B)
	YbDivS, err := vector.Hadamard(bDivS, Hpf)
	if err != nil {
		return false, err
	}

	c3, err := vector.Exp(YbDivS, HVec, len(HVec), 1)
	if err != nil {
		return false, err
	}
	var abw ristretto.Scalar
	abw.Mul(&ipproof.A, &ipproof.B)
	abw.Mul(&abw, &w)

	var c4 ristretto.Point
	c4.ScalarMult(&G, &abw)

	c5, err := vector.Exp(uSq, ipproof.L, len(ipproof.L), 1)
	if err != nil {
		return false, err
	}

	c6, err := vector.Exp(uInvSq, ipproof.R, len(ipproof.R), 1)
	if err != nil {
		return false, err
	}

	var sum ristretto.Point
	sum.SetZero()
	sum.Add(&P, &c1)
	sum.Sub(&sum, &c2)
	sum.Sub(&sum, &c3)
	sum.Sub(&sum, &c4)
	sum.Add(&sum, &c5)
	sum.Add(&sum, &c6)

	var zero ristretto.Point
	zero.SetZero()

	ok := sum.Equals(&zero)

	if !ok {
		return false, errors.New("sum is not zero")
	}

	return true, nil

}
