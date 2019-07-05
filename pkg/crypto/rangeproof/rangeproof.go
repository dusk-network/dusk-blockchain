package rangeproof

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/pkg/errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/fiatshamir"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/innerproduct"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/pedersen"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/vector"
)

// N is number of bits in range
// So amount will be between 0...2^(N-1)
const N = 64

// M is the number of outputs for one bulletproof
var M = 1

// M is the maximum number of values allowed per rangeproof
const maxM = 16

// Proof is the constructed BulletProof
type Proof struct {
	V        []pedersen.Commitment // Curve points 32 bytes
	Blinders []ristretto.Scalar
	A        ristretto.Point // Curve point 32 bytes
	S        ristretto.Point // Curve point 32 bytes
	T1       ristretto.Point // Curve point 32 bytes
	T2       ristretto.Point // Curve point 32 bytes

	taux ristretto.Scalar //scalar
	mu   ristretto.Scalar //scalar
	t    ristretto.Scalar

	IPProof *innerproduct.Proof
}

// Prove will take a set of scalars as a parameter and prove that it is [0, 2^N)
func Prove(v []ristretto.Scalar, debug bool) (Proof, error) {

	if len(v) < 1 {
		return Proof{}, errors.New("length of slice v is zero")
	}

	M = len(v)
	if M > maxM {
		return Proof{}, fmt.Errorf("maximum amount of values must be less than %d", maxM)
	}

	// Pad zero values until we have power of two
	padAmount := innerproduct.DiffNextPow2(uint32(M))
	M = M + int(padAmount)
	for i := uint32(0); i < padAmount; i++ {
		var zeroScalar ristretto.Scalar
		zeroScalar.SetZero()
		v = append(v, zeroScalar)
	}

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

// Verify takes a bullet proof and returns true only if the proof was valid
func Verify(p Proof) (bool, error) {

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(uint32(N * M))

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(uint32(N * M))

	G := ped.BaseVector.Bases
	H := ped2.BaseVector.Bases

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

	return megacheckWithC(p.IPProof, p.mu, x, y, z, p.t, p.taux, w, p.A, ped.BasePoint, ped.BlindPoint, p.S, p.T1, p.T2, G, H, p.V)
}

func megacheckWithC(ipproof *innerproduct.Proof, mu, x, y, z, t, taux, w ristretto.Scalar, A, G, H, S, T1, T2 ristretto.Point, GVec, HVec []ristretto.Point, V []pedersen.Commitment) (bool, error) {

	var c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11 ristretto.Point

	var c ristretto.Scalar
	c.Rand()

	uSq, uInvSq, s := ipproof.VerifScalars()
	sInv := make([]ristretto.Scalar, len(s))
	copy(sInv, s)

	// reverse s
	for i, j := 0, len(sInv)-1; i < j; i, j = i+1, j-1 {
		sInv[i], sInv[j] = sInv[j], sInv[i]
	}

	// g vector scalars : as + z points : G
	as := vector.MulScalar(s, ipproof.A)
	g := vector.AddScalar(as, z)
	g = vector.MulScalar(g, c)

	c1, err := vector.Exp(g, GVec, len(GVec), 1)
	if err != nil {
		return false, err
	}

	// h vector scalars : y Had (bsInv - zM2N) - z points : H
	bs := vector.MulScalar(sInv, ipproof.B)
	zAnd2 := sumZMTwoN(z)
	h, err := vector.Sub(bs, zAnd2)
	if err != nil {
		return false, errors.Wrap(err, "[h1]")
	}

	var yinv ristretto.Scalar
	yinv.Inverse(&y)
	Hpf := vector.ScalarPowers(yinv, uint32(N*M))

	h, err = vector.Hadamard(h, Hpf)
	if err != nil {
		return false, errors.Wrap(err, "[h2]")
	}
	h = vector.SubScalar(h, z)
	h = vector.MulScalar(h, c)

	c2, err = vector.Exp(h, HVec, len(HVec), 1)
	if err != nil {
		return false, err
	}

	// G basepoint gbp : (c * w(ab-t)) + t-D(y,z) point : G
	delta := computeDelta(y, z, N, uint32(M))
	var tMinusDelta ristretto.Scalar
	tMinusDelta.Sub(&t, &delta)

	var abMinusT ristretto.Scalar
	abMinusT.Mul(&ipproof.A, &ipproof.B)
	abMinusT.Sub(&abMinusT, &t)

	var cw ristretto.Scalar
	cw.Mul(&c, &w)

	var gBP ristretto.Scalar
	gBP.MulAdd(&cw, &abMinusT, &tMinusDelta)

	c3.ScalarMult(&G, &gBP)

	// H basepoint hbp : c * mu + taux point: H
	var cmu ristretto.Scalar
	cmu.Mul(&mu, &c)

	var hBP ristretto.Scalar
	hBP.Add(&cmu, &taux)

	c4.ScalarMult(&H, &hBP)

	// scalar :c point: A
	c5.ScalarMult(&A, &c)

	//  scalar: cx point : S
	var cx ristretto.Scalar
	cx.Mul(&c, &x)
	c6.ScalarMult(&S, &cx)

	// scalar: uSq challenges  points: Lj
	c7, err = vector.Exp(uSq, ipproof.L, len(ipproof.L), 1)
	if err != nil {
		return false, err
	}
	c7.PublicScalarMult(&c7, &c)

	// scalar : uInvSq challenges points: Rj
	c8, err = vector.Exp(uInvSq, ipproof.R, len(ipproof.R), 1)
	if err != nil {
		return false, err
	}
	c8.PublicScalarMult(&c8, &c)

	// scalar: z_j+2  points: Vj
	zM := vector.ScalarPowers(z, uint32(M))
	var zSq ristretto.Scalar
	zSq.Square(&z)
	zM = vector.MulScalar(zM, zSq)
	c9.SetZero()
	for i := range zM {
		var temp ristretto.Point
		temp.PublicScalarMult(&V[i].Value, &zM[i])
		c9.Add(&c9, &temp)
	}

	// scalar : x point: T1
	c10.PublicScalarMult(&T1, &x)

	// scalar : xSq point: T2
	var xSq ristretto.Scalar
	xSq.Square(&x)
	c11.PublicScalarMult(&T2, &xSq)

	var sum ristretto.Point
	sum.SetZero()
	sum.Add(&c1, &c2)
	sum.Add(&sum, &c3)
	sum.Add(&sum, &c4)
	sum.Sub(&sum, &c5)
	sum.Sub(&sum, &c6)
	sum.Sub(&sum, &c7)
	sum.Sub(&sum, &c8)
	sum.Sub(&sum, &c9)
	sum.Sub(&sum, &c10)
	sum.Sub(&sum, &c11)

	var zero ristretto.Point
	zero.SetZero()

	ok := zero.Equals(&sum)
	if !ok {
		return false, errors.New("megacheck failed")
	}

	return true, nil
}

func (p *Proof) Encode(w io.Writer, includeCommits bool) error {

	if includeCommits {
		err := pedersen.EncodeCommitments(w, p.V)
		if err != nil {
			return err
		}
	}

	err := binary.Write(w, binary.BigEndian, p.A.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.S.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.T1.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.T2.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.taux.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.mu.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.t.Bytes())
	if err != nil {
		return err
	}
	return p.IPProof.Encode(w)
}

func (p *Proof) Decode(r io.Reader, includeCommits bool) error {

	if p == nil {
		return errors.New("struct is nil")
	}

	if includeCommits {
		comms, err := pedersen.DecodeCommitments(r)
		if err != nil {
			return err
		}
		p.V = comms
	}

	err := readerToPoint(r, &p.A)
	if err != nil {
		return err
	}
	err = readerToPoint(r, &p.S)
	if err != nil {
		return err
	}
	err = readerToPoint(r, &p.T1)
	if err != nil {
		return err
	}
	err = readerToPoint(r, &p.T2)
	if err != nil {
		return err
	}
	err = readerToScalar(r, &p.taux)
	if err != nil {
		return err
	}
	err = readerToScalar(r, &p.mu)
	if err != nil {
		return err
	}
	err = readerToScalar(r, &p.t)
	if err != nil {
		return err
	}
	p.IPProof = &innerproduct.Proof{}
	return p.IPProof.Decode(r)
}

func (p *Proof) Equals(other Proof, includeCommits bool) bool {
	if len(p.V) != len(other.V) && includeCommits {
		return false
	}

	for i := range p.V {
		ok := p.V[i].EqualValue(other.V[i])
		if !ok {
			return ok
		}
	}

	ok := p.A.Equals(&other.A)
	if !ok {
		return ok
	}
	ok = p.S.Equals(&other.S)
	if !ok {
		return ok
	}
	ok = p.T1.Equals(&other.T1)
	if !ok {
		return ok
	}
	ok = p.T2.Equals(&other.T2)
	if !ok {
		return ok
	}
	ok = p.taux.Equals(&other.taux)
	if !ok {
		return ok
	}
	ok = p.mu.Equals(&other.mu)
	if !ok {
		return ok
	}
	ok = p.t.Equals(&other.t)
	if !ok {
		return ok
	}
	return true
	return p.IPProof.Equals(*other.IPProof)
}

func readerToPoint(r io.Reader, p *ristretto.Point) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	ok := p.SetBytes(&x)
	if !ok {
		return errors.New("point not encodable")
	}
	return nil
}
func readerToScalar(r io.Reader, s *ristretto.Scalar) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	s.SetBytes(&x)
	return nil
}
