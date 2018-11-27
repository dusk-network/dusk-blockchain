package rangeproof

import (
	"errors"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/pedersen"
	"github.com/toghrulmaharramov/dusk-go/rangeproof/vector"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// N is number of bits in range
// So amount will be between 0..2^(N-1)
const N = 64

// M is the maximum number of outputs
// for one bulletproof
const M = 1

// Proof is the constructed BulletProof
// XXX: For now we are not including the aggregated case, or the inner product proof to shorten the transmitted variables
type Proof struct {
	V  ristretto.Point // Curve points 32 bytes, multi-proof but limit size to one
	A  ristretto.Point // Curve point 32 bytes
	S  ristretto.Point // Curve point 32 bytes
	T1 ristretto.Point // Curve point 32 bytes
	T2 ristretto.Point // Curve point 32 bytes

	taux ristretto.Scalar //scalar
	mu   ristretto.Scalar //scalar
	t    ristretto.Scalar

	l []ristretto.Scalar // []scalar
	r []ristretto.Scalar // []scalar
}

// Prove will take a scalar as a parameter and
// using zero knowledge, prove that it is [0, 2^N)
func Prove(v ristretto.Scalar, debug bool) (Proof, error) {

	ped := pedersen.New([]byte("dusk.BulletProof.vec1"))

	// Hash for Fiat-Shamir
	hs := hashCacher{cache: []byte{}}

	// compute commmitment to v
	V := ped.CommitToScalars(v)

	// update Fiat-Shamir
	hs.Append(V.Value.Bytes())

	// Compute Bitcommits aL and aR to v
	BitCommitment := BitCommit(v.BigInt())

	// Compute A
	A := computeA(ped, BitCommitment.AL, BitCommitment.AR)

	// Compute S
	S, sL, sR := computeS(ped)

	// update Fiat-Shamir
	hs.Append(A.Value.Bytes(), S.Value.Bytes())

	// compute y and z
	y, z := computeYAndZ(hs)

	// compute polynomial
	poly := computePoly(BitCommitment.AL, BitCommitment.AR, sL, sR, y, z, v)

	// Compute T1 and T2
	T1 := ped.CommitToScalars(poly.t1)
	T2 := ped.CommitToScalars(poly.t2)

	// update Fiat-Shamir
	hs.Append(z.Bytes(), T1.Value.Bytes(), T2.Value.Bytes())

	// compute x
	x := computeX(hs)

	// compute taux which is just the polynomial for the blinding factors at a point x
	taux := computeTaux(x, z, V.BlindingFactor, T1.BlindingFactor, T2.BlindingFactor)

	// compute mu
	mu := computeMu(x, A.BlindingFactor, S.BlindingFactor)

	// compute l dot r
	l := poly.computeL(x)
	r := poly.computeR(x)
	t, _ := innerProduct(l, r)

	// DEBUG
	if debug {
		err := debugProve(v, x, y, z, l, r, BitCommitment.AL, BitCommitment.AR, sL, sR)
		if err != nil {
			return Proof{}, err
		}
		// DEBUG T0
		testT0 := debugT0(BitCommitment.AL, BitCommitment.AR, y, z)
		if !testT0.Equals(&poly.t0) {
			return Proof{}, errors.New("[Prove]: Test t0 value does not match the value calculated from the polynomial")
		}

		polyt0 := poly.computeT0(y, z, v)
		if !polyt0.Equals(&poly.t0) {
			return Proof{}, errors.New("[Prove]: t0 value from delta function, does not match the polynomial t0 value(Correct)")
		}

		tPoly := poly.eval(x)
		if !t.Equals(&tPoly) {
			return Proof{}, errors.New("[Prove]: The t value computed from the t-poly, does not match the t value computed from the inner product of l and r")
		}

		// Debug aL and aR
		err = BitCommitment.Debug(v.BigInt())
		if err != nil {
			return Proof{}, err
		}
	}
	// End DEBUG

	// TODO: calculate inner product proof

	return Proof{
		V:    V.Value,
		A:    A.Value,
		S:    S.Value,
		T1:   T1.Value,
		T2:   T2.Value,
		l:    l,
		r:    r,
		t:    t,
		taux: taux,
		mu:   mu,
	}, nil
}

// A = kH + aL*G + aR * H
func computeA(ped *pedersen.Pedersen, aL, aR [N]ristretto.Scalar) pedersen.Commitment {

	cA := ped.CommitToVectors(aL[:], aR[:])

	return cA
}

// S = kH + sL*G + sR * H
func computeS(ped *pedersen.Pedersen) (pedersen.Commitment, [N]ristretto.Scalar, [N]ristretto.Scalar) {

	var sL, sR [N]ristretto.Scalar
	for i := 0; i < N; i++ {
		var randA ristretto.Scalar
		randA.Rand()
		sL[i] = randA

		var randB ristretto.Scalar
		randB.Rand()
		sR[i] = randB
	}

	cS := ped.CommitToVectors(sL[:], sR[:])

	return cS, sL, sR
}

func computeYAndZ(hs hashCacher) (ristretto.Scalar, ristretto.Scalar) {

	var y ristretto.Scalar
	y.Derive(hs.Result())

	var z ristretto.Scalar
	z.Derive(y.Bytes())

	return y, z
}
func computeX(hs hashCacher) ristretto.Scalar {
	var x ristretto.Scalar
	x.Derive(hs.Result())
	return x
}

// compute polynomial for blinding factors l61
// N.B. tau1 means tau superscript 1
func computeTaux(x, z, vBlind, t1Blind, t2Blind ristretto.Scalar) ristretto.Scalar {
	tau1X := t1Blind.Mul(&x, &t1Blind)

	var xsq ristretto.Scalar
	xsq.Square(&x)

	tau2Xsq := t2Blind.Mul(&xsq, &t2Blind)

	var zSq ristretto.Scalar
	zSq.Square(&z)

	var zSqvBlind ristretto.Scalar
	zSqvBlind.Mul(&zSq, &vBlind)

	var res ristretto.Scalar
	res.Add(tau1X, tau2Xsq)
	res.Add(&res, &zSqvBlind)

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
func computeP(Blind, A, S ristretto.Point, mu, x, y, z ristretto.Scalar) ristretto.Point {

	var P ristretto.Point
	P.SetZero()

	var xS ristretto.Point
	xS.ScalarMult(&S, &x)

	Si := computeSi(y, z)

	// Add - mu * blindingFactor
	var muB ristretto.Point
	muB.ScalarMult(&Blind, &mu)
	muB.Neg(&muB)

	P.Add(&A, &xS)
	P.Add(&P, &muB)
	P.Add(&P, &Si)

	return P
}

// P = A + xS + Si(y,z)
func computeSi(y, z ristretto.Scalar) ristretto.Point {

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(65)

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(64)

	Hprime := computeHprime(ped2.BaseVector.Bases, y)

	yN := vector.ScalarPowers(y, N)

	zYn := vector.MulScalar(yN, z)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vector.ScalarPowers(two, N)

	zsq2n := vector.MulScalar(twoN, zsq)

	leftIP, _ := vector.Add(zsq2n, zYn)

	leftSi, _ := vector.Exp(leftIP, Hprime, N, M)

	// -z<1, G>
	var minusZ ristretto.Scalar
	minusZ.Neg(&z)

	G := ped.BaseVector.Bases[1:]

	rightSi, _ := vector.Exp(vector.FromScalar(minusZ, N), G, N, M)

	var Si ristretto.Point
	Si.SetZero()

	Si.Add(&leftSi, &rightSi)

	return Si
}

// computeHprime will take a a slice of points H, with a scalar y
// and return a slice of points Hprime,  such that Hprime = y^-n * H
func computeHprime(H []ristretto.Point, y ristretto.Scalar) []ristretto.Point {
	Hprimes := make([]ristretto.Point, len(H))

	yInv := y.Inverse()
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
func computeLGRH(y, mu ristretto.Scalar, l, r []ristretto.Scalar) ristretto.Point {

	var P ristretto.Point
	P.SetZero()

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(65)

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(64)

	Hprime := computeHprime(ped2.BaseVector.Bases, y)
	G := ped.BaseVector.Bases[1:]

	rH, _ := vector.Exp(r, Hprime, N, M)
	lG, _ := vector.Exp(l, G, N, M)

	P.Add(&lG, &rH)
	return P
}

// Verify takes a bullet proof and
// returns true only if the proof was valid
func Verify(p Proof) (bool, error) {

	ped := pedersen.New([]byte("dusk.BulletProof.vec1"))
	ped.BaseVector.Compute(2)

	if len(p.l) != len(p.r) {
		return false, errors.New("[Verify]: Sizes of l and r do not match")
	}

	if len(p.l) <= 0 {
		return false, errors.New("[Verify]: size of l or r cannot be zero; empty proof")
	}

	// Reconstruct the challenges
	hs := hashCacher{[]byte{}}
	hs.Append(p.V.Bytes(), p.A.Bytes(), p.S.Bytes())
	y, z := computeYAndZ(hs)

	hs.Append(z.Bytes(), p.T1.Bytes(), p.T2.Bytes())
	x := computeX(hs)

	// compute l dot r
	t, _ := innerProduct(p.l, p.r)

	// compute tG + tauH = Z^2 * V + delta(y,z) -- Prove t0 is correct
	_, err := VerifyT0(x, y, z, t, p.taux, ped.BaseVector.Bases[0], ped.BaseVector.Bases[1], p.V, p.T1, p.T2)
	if err != nil {
		return false, err
	}

	// prove l(x) and r(x) is correct

	Pleft := computeLGRH(y, p.mu, p.l, p.r)

	PRight := computeP(ped.BaseVector.Bases[0], p.A, p.S, p.mu, x, y, z)

	if !Pleft.Equals(&PRight) {
		return false, errors.New("[Verify]: Proof for l(x),r(x) is wrong")
	}

	return true, nil
}

func VerifyT0(x, y, z, t, taux ristretto.Scalar, Blind, G1, V, T1, T2 ristretto.Point) (bool, error) {
	// LHS
	var tG ristretto.Point
	tG.ScalarMult(&G1, &t)
	var tauH ristretto.Point
	tauH.ScalarMult(&Blind, &taux)

	var LHS ristretto.Point
	LHS.Add(&tG, &tauH)

	// RHS
	var cT1x ristretto.Point
	cT1x.ScalarMult(&T1, &x)

	var xsq ristretto.Scalar
	xsq.Square(&x)

	var cT2xsq ristretto.Point
	cT2xsq.ScalarMult(&T2, &xsq)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	var zsqV ristretto.Point
	zsqV.ScalarMult(&V, &zsq)

	var deltaG ristretto.Point
	delta := computeDelta(y, z)

	deltaG.ScalarMult(&G1, &delta)

	var RHS1 ristretto.Point
	var RHS2 ristretto.Point
	var RHS ristretto.Point
	RHS1.Add(&zsqV, &deltaG)
	RHS2.Add(&cT2xsq, &cT1x)
	RHS.Add(&RHS1, &RHS2)

	if !LHS.Equals(&RHS) {
		return false, errors.New("[Verify]: LHS != RHS; proof that t0 is correct is wrong")
	}
	return true, nil
}
