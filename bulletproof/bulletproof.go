// Package bulletproof is a reference implementation of the bulletproof used at monero
// with some optimisations from the basic ed25519 curve functions in golang
// Reference found here: https://github.com/monero-project/monero/blob/master/src/ringct/bulletproofs.cc
package bulletproof

import (
	"errors"

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

// vecExp takes two scalars and returns a vector commitment C
// the calculation goes as follows C = aG + bH
// Where H is the Hash of the Generator G
func vecExp(a ristretto.Scalar, b ristretto.Scalar) (ristretto.Point, error) {
	result := ristretto.Point{} // defaults to zero

	if len(a) != len(b) {
		return result, errors.New("length of scalar a does not equal length of scalar b")
	}

	if len(a) <= N*M {
		return result, errors.New("length of scalar a is not less than N*M")
	}

	// aG+bH
	var aG ristretto.Point
	aG.ScalarMultBase(&a)

	var bH ristretto.Point
	bH.SetBytes(&H)
	bH.ScalarMult(&bH, &b)

	result.Add(&aG, &bH)

	return result, nil
}

// Given a scalar, construct a vector of powers
func vecPowers(a ristretto.Scalar, n uint8) []ristretto.Scalar {

	var res []ristretto.Scalar

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

	var res []ristretto.Scalar

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

// Prove takes an amount and a mask and constructs a bulletproof
// Given a set of values v (0..2^N-1) and masks gamma, construct a range proof
func Prove(amount, mask ristretto.Scalar) Proof {

	return Proof{}
}

// Verify takes a bullet proof and
// returns true only if the proof was valid
func Verify(p Proof) bool {
	return false
}
