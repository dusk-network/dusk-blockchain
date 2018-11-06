package rangeproof

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// N is number of bits in range
// So amount will be between 0..2^(N-1)
const N = 64

// M is the maximum number of outputs
// for one bulletproof
const M = 1

var zeroK ristretto.Scalar

// curve base points
var (
	// TODO:For Gi and Hi, we do not have an API with Ristretto for it
	// use the CompletedPoints for now.

	Gi = [N]ristretto.Point{}
	Hi = [N]ristretto.Point{}
)

// Proof is the constructed BulletProof
type Proof struct {
	V  []ristretto.Point // Curve points 32 bytes, multi-proof but limit size to one
	A  ristretto.Point   // Curve point 32 bytes
	S  ristretto.Point   // Curve point 32 bytes
	T1 ristretto.Point   // Curve point 32 bytes
	T2 ristretto.Point   // Curve point 32 bytes

	taux ristretto.Scalar //scalar
	mu   ristretto.Scalar //scalar

	l []ristretto.Scalar // []scalar
	r []ristretto.Scalar // []scalar
}

// vecExp takes two scalar arrays and returns a vector commitment C
// the calculation goes as follows C = aG + bH
// Where H is the Hash of the Generator G
func vecExp(a, b []ristretto.Scalar) (ristretto.Point, error) {
	result := ristretto.Point{} // defaults to zero

	if len(a) != len(b) {
		return result, errors.New("length of scalar a does not equal length of scalar b")
	}

	if len(a) < N*M {
		return result, errors.New("length of scalar a is not less than N*M")
	}

	// aG+bH

	for i := range a {
		var aG ristretto.Point
		aG.ScalarMultBase(&a[i])

		var bH ristretto.Point
		bH.SetBytes(&H)
		bH.ScalarMult(&bH, &b[i])

		result.Add(&aG, &bH)
	}

	return result, nil
}

// Given a scalar, construct a vector of powers
func vecPowers(a ristretto.Scalar, n uint8) []ristretto.Scalar {

	res := make([]ristretto.Scalar, n)

	if n == 0 {
		return res
	}

	// id
	var k ristretto.Scalar
	k.SetBytes(&Identity)
	res[0] = k

	if n == 1 {
		return res
	}
	res[1] = a

	for i := uint8(2); i < n; i++ {
		res[i].Add(&res[i-1], &a)
	}

	return res
}

// Given two scalar arrays, construct the inner product
func innerProduct(a, b []ristretto.Scalar) (ristretto.Scalar, error) {

	var res ristretto.Scalar

	if len(a) != len(b) {
		return res, errors.New("Length of a does not equal length of b")
	}

	for i := 0; i < len(a); i++ {
		res.MulAdd(&a[i], &b[i], &res)
	}

	return res, nil
}

// Given two scalar arrays, construct the Hadamard product
func hadamard(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {

	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal length of b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}
	return res, nil
}

// Given two curvepoint arrays, construct the Hadamard product
func hadamard2(a, b []ristretto.Point) ([]ristretto.Point, error) {

	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal length of b")
	}

	res := make([]ristretto.Point, len(a))
	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}

	return res, nil
}

func vecAdd(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal b")
	}

	var res []ristretto.Scalar

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}

	return res, nil
}

func vecSub(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Sub(&a[i], &b[i])
	}

	return res, nil
}

func vecScal(a ristretto.Scalar, b []ristretto.Scalar) ([]ristretto.Scalar, error) {

	var res []ristretto.Scalar

	for i := 0; i < len(a); i++ {
		res[i].Mul(&b[i], &a)
	}

	return res, nil
}

// Given a scalar, return the sum of its powers from 0 to n-1
func vecPowerSum(a ristretto.Scalar, n uint64) ristretto.Scalar {

	var res ristretto.Scalar
	res.SetZero()

	if n == 0 {
		return res
	}

	res.SetOne()

	if n == 1 {
		return res
	}

	prev := a

	for i := uint64(1); i < n; i++ {
		if i > 1 {
			prev.Mul(&prev, &a)
		}
		res.Add(&res, &prev)
	}

	return res
}

func scInv(a ristretto.Scalar) *ristretto.Scalar {
	a.Inverse(&a)
	return &a
}

// hashCacheMash will take an existing hashCache
// add n amount of byte slices
// then return it all using hashToScalar
func hashCacheMash(hashCache ristretto.Scalar, mashes ...[]byte) ristretto.Scalar {

	data := []byte{}
	data = append(data, hashCache.Bytes()...)

	for _, mash := range mashes {
		data = append(data, mash...)
	}

	var s ristretto.Scalar
	s.Derive(data)
	return s
}

// Prove takes an amount and a mask and constructs a bulletproof
// Given a set of values v (0..2^N-1) and masks gamma, construct a range proof
// Will do single proof for now, then multiple proof later on
func Prove(v, gamma ristretto.Scalar) (Proof, error) {

	// TODO : check v is in correct range
	// will produce an incorrect proof anyways
	// but will be fast to exit quickly

	// V = v *H + gamma * G
	var V ristretto.Point
	var vH ristretto.Point
	vH.SetBytes(&H)
	vH.ScalarMult(&vH, &v)
	var gamG ristretto.Point
	gamG.ScalarMultBase(&gamma)
	V.Add(&vH, &gamG)

	// This hash is updated for Fiat-Shamir throughout the proof
	var hashCache ristretto.Scalar
	hashCache.Derive(V.Bytes())

	// PAPER LINES 36-37
	var aL, aR [N]ristretto.Scalar

	tempV := v.BigInt()

	// zero and one scalars
	var zero ristretto.Scalar
	zero.SetZero()
	var one ristretto.Scalar
	one.SetOne()

	for i := N - 1; i >= 0; i-- {
		var basePow, e = big.NewInt(2), big.NewInt(int64(i))
		basePow.Exp(basePow, e, nil)

		tV := big.NewInt(0) // we don't want to edit the tempV value, so we have this dummy val

		if (tV.Div(tempV, basePow)).Cmp(big.NewInt(0)) == 0 {
			aL[i] = zero
		} else {

			aL[i] = one
			tempV.Sub(tempV, basePow)
		}
		var aRi ristretto.Scalar
		aRi.Sub(&aL[i], &one)
		aR[i] = aRi
	}

	// DEBUG : Test to ensure we recover the values
	testAL := big.NewInt(0)
	testAR := big.NewInt(0)

	for i := 0; i < N; i++ {

		var basePow, e = big.NewInt(2), big.NewInt(int64(i))
		basePow.Exp(basePow, e, nil)

		if aL[i].Equals(&one) {
			testAL = testAL.Add(testAL, basePow)
		}
		if aR[i].Equals(&zero) {
			testAR = testAR.Add(testAR, basePow)
		}
	}

	if testAL.Cmp(v.BigInt()) != 0 {
		return Proof{}, errors.New("Wrong Value for AL")
	}

	if testAR.Cmp(v.BigInt()) != 0 {
		return Proof{}, errors.New("Wrong Value for AL")
	}

	// PAPER LINES 38-39
	var alpha ristretto.Scalar
	alpha.Rand()

	var alpG ristretto.Point
	alpG.ScalarMultBase(&alpha)

	A, err := vecExp(aL[:], aR[:])
	if err != nil {
		fmt.Println(len(aL), len(aR), M*N)
		return Proof{}, err
	}
	A.Add(&A, &alpG)

	// PAPER LINES 40-42
	var sL, sR [N]ristretto.Scalar
	for i := 0; i < N; i++ {
		var randA ristretto.Scalar
		randA.Rand()
		sL[i] = randA

		var randB ristretto.Scalar
		randB.Rand()
		sR[i] = randB
	}

	var rho ristretto.Scalar
	rho.Rand()
	S, err := vecExp(sL[:], sR[:])
	if err != nil {
		return Proof{}, err
	}
	var rhoG ristretto.Point
	rhoG.ScalarMultBase(&rho)
	S.Add(&S, &rhoG)

	// PAPER LINES 43-45
	hashCache = hashCacheMash(hashCache, A.Bytes(), S.Bytes())
	y := hashCache
	hashCache.Derive(hashCache.Bytes())
	z := hashCache

	var t0, tempZ ristretto.Scalar

	tempZ = z
	vp1 := vecPowers(one, N)
	vpy := vecPowers(y, N)
	if err != nil {
		return Proof{}, err
	}
	ip, err := innerProduct(vp1, vpy)
	if err != nil {
		return Proof{}, err
	}
	t0.Add(&t0, tempZ.Mul(&z, &ip))
	tempZ = z
	t0.Add(&t0, tempZ.Square(&tempZ).Mul(&tempZ, &v))

	k := computeK(&y, &z)
	t0.Add(&t0, &k)

	// DEBUG: Test the value of t0 has the correct form

	hd, err := hadamard(aR[:], vecPowers(y, N))
	if err != nil {
		return Proof{}, err
	}
	ipt0, err := innerProduct(aL[:], hd)
	if err != nil {
		return Proof{}, err
	}
	test_t0 := ipt0

	vS, err := vecSub(aL[:], aR[:])
	if err != nil {
		return Proof{}, err
	}
	ipt1, err := innerProduct(vS, vecPowers(y, N))
	if err != nil {
		return Proof{}, err
	}
	test_t0.Add(&test_t0, &ipt1)

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	ipt2, err := innerProduct(vecPowers(two, N), aL[:])
	test_t0.Add(&test_t0, &ipt2)

	test_t0.Add(&test_t0, &k)
	if test_t0.Equals(&t0) {
		return Proof{}, errors.New("Wrong value for t0")
	}
	return Proof{}, nil
}

func computeK(y, z *ristretto.Scalar) ristretto.Scalar {
	var res ristretto.Scalar
	res.SetZero()

	var one ristretto.Scalar
	one.SetOne()
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	vpy := vecPowers(*y, N)
	vp1 := vecPowers(one, N)
	ip, err := innerProduct(vp1, vpy)
	if err != nil {
		return ristretto.Scalar{}
	}

	var tempZ ristretto.Scalar
	tempZ = *z

	res.Sub(&res, tempZ.Square(&tempZ).Mul(&tempZ, &ip))

	vp2 := vecPowers(two, N)
	ip, err = innerProduct(vp1, vp2)
	if err != nil {
		return ristretto.Scalar{}
	}

	tempZ = *z

	for i := 0; i < 3; i++ { //cubed
		tempZ.Mul(&tempZ, &tempZ)
	}

	res.Sub(&res, tempZ.Mul(&tempZ, &ip))
	return res
}

// Verify takes a bullet proof and
// returns true only if the proof was valid
func Verify(p Proof) bool {
	return false
}
