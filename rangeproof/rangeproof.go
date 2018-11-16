package rangeproof

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/rangeproof/pedersen"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// N is number of bits in range
// So amount will be between 0..2^(N-1)
const N = 64

// M is the maximum number of outputs
// for one bulletproof
const M = 1

// Proof is the constructed BulletProof
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

func Prove(v ristretto.Scalar) (Proof, error) {

	ped := pedersen.New([]byte("dusk.BulletProof.vec1")) // XXX: this will set the generator, should we use the standard base

	// Hash for Fiat-Shamir
	hs := hashCacher{cache: []byte{}}

	// compute commmitment to v
	cV := ped.CommitToScalars(v)

	// update Fiat-Shamir
	hs.Append(cV.Value.Bytes())

	// Compute Bitcomments aL and aR to v
	BitCommitment := BitCommit(v.BigInt())
	ok, err := BitCommitment.Ensure(v.BigInt())
	if !ok || err != nil {
		return Proof{}, errors.New("Error Committing value v to Bit slices aL and aR ")
	}

	// Compute A
	cA := computeA(ped, BitCommitment.AL, BitCommitment.AR)

	// Compute S
	cS, sL, sR := computeS(ped)

	// update Fiat-Shamir
	hs.Append(cA.Value.Bytes())
	hs.Append(cS.Value.Bytes())

	// compute y and z
	y, z := computeYAndZ(hs)

	// compute polynomial
	poly := computePoly(BitCommitment.AL, BitCommitment.AR, sL, sR, y, z, v)

	// Compute T1 and T2
	cT1 := ped.CommitToScalars(poly.t1)
	cT2 := ped.CommitToScalars(poly.t2)

	// update Fiat-Shamir
	hs.Append(z.Bytes())
	hs.Append(cT1.Value.Bytes())
	hs.Append(cT2.Value.Bytes())

	// compute x
	x := computeX(hs)

	// compute taux which is just the polynomial for the blinding factors at a point x
	taux := computeTaux(x, z, cV.BlindingFactor, cT1.BlindingFactor, cT2.BlindingFactor)

	// TODO: compute mu

	// compute l dot r
	l := poly.computeL(x)
	r := poly.computeR(x)
	t, _ := innerProduct(l, r)

	testT0 := testT0(BitCommitment.AL, BitCommitment.AR, y, z)
	if !testT0.Equals(&poly.t0) {
		return Proof{}, errors.New("Test t0 value does not match the value calculated from the polynomial")
	}

	polyt0 := poly.computeT0(y, z, v)
	if !polyt0.Equals(&poly.t0) {
		return Proof{}, errors.New("t0 value from delta function, does not match the polynomial t0 value(Correct)")
	}

	tPoly := poly.eval(x)
	if !t.Equals(&tPoly) {
		return Proof{}, errors.New("The t value computed from the t-poly, does not match the t value computed from the inner product of l and r")
	}

	// TODO: calculate inner product proof

	return Proof{
		V:    cV.Value,
		A:    cA.Value,
		S:    cS.Value,
		T1:   cT1.Value,
		T2:   cT2.Value,
		l:    l,
		r:    r,
		t:    t,
		taux: taux,
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

// Verify takes a bullet proof and
// returns true only if the proof was valid
func Verify(p Proof) (bool, error) {

	ped := pedersen.New([]byte("dusk.BulletProof.vec1"))
	ped.BaseVector.Compute(2)

	if len(p.l) != len(p.r) {
		return false, errors.New("Sizes of l and r do not match")
	}

	if len(p.l) <= 0 {
		fmt.Println("size of l is ", p.l)
		return false, errors.New("size of l or r cannot be zero; empty proof")
	}

	// Reconstruct the challenges
	hs := hashCacher{[]byte{}}
	hs.Append(p.V.Bytes())
	hs.Append(p.A.Bytes())
	hs.Append(p.S.Bytes())

	y, z := computeYAndZ(hs)
	_ = y
	_ = z

	hs.Append(z.Bytes())
	hs.Append(p.T1.Bytes())
	hs.Append(p.T2.Bytes())

	x := computeX(hs)

	// compute l dot r
	t, _ := innerProduct(p.l, p.r)

	// compute tG + tauH = Z^2 * V + delta(y,z) -- Prove t0 is correct

	// LHS
	var tG ristretto.Point
	tG.ScalarMult(&ped.BaseVector.Bases[1], &t)
	var tauH ristretto.Point
	tauH.ScalarMult(&ped.BaseVector.Bases[0], &p.taux)

	var LHS ristretto.Point
	LHS.Add(&tG, &tauH)

	// RHS
	var cT1x ristretto.Point
	cT1x.ScalarMult(&p.T1, &x)

	var xsq ristretto.Scalar
	xsq.Square(&x)

	var cT2xsq ristretto.Point
	cT2xsq.ScalarMult(&p.T2, &xsq)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	var zsqV ristretto.Point
	zsqV.ScalarMult(&p.V, &zsq)

	var deltaG ristretto.Point
	delta, err := computeDelta(y, z)
	if err != nil {
		return false, err
	}

	deltaG.ScalarMult(&ped.BaseVector.Bases[0], &delta)

	var RHS1 ristretto.Point
	var RHS2 ristretto.Point
	var RHS ristretto.Point
	RHS1.Add(&zsqV, &deltaG)
	RHS2.Add(&cT2xsq, &cT1x)
	RHS.Add(&RHS1, &RHS2)

	if !LHS.Equals(&RHS) {
		fmt.Println(LHS, RHS)
		return false, errors.New("LHS != RHS; proof that t0 is correct is wrong")
	}

	// prove l(x) and r(x) is correct

	return true, nil
}

// DEBUG

func testT0(aL, aR [N]ristretto.Scalar, y, z ristretto.Scalar) ristretto.Scalar {

	aLMinusZ, _ := vecSubScal(aL[:], z)

	aRPlusZ, _ := vecAddScal(aR[:], z)

	yN := vecPowers(y, N)

	hada, _ := hadamard(yN, aRPlusZ)

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vecPowers(two, N)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	zsqMul2n, _ := vecScal(twoN, zsq)

	rightIP, _ := vecAdd(zsqMul2n, hada)

	iP, _ := innerProduct(aLMinusZ, rightIP)

	return iP
}
