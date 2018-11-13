package rangeproof

import (
	"errors"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func vecAdd(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}

	return res, nil
}
func vecAddScal(a []ristretto.Scalar, b ristretto.Scalar) ([]ristretto.Scalar, error) {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b)
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
func vecSubScal(a []ristretto.Scalar, b ristretto.Scalar) ([]ristretto.Scalar, error) {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Sub(&a[i], &b)
	}

	return res, nil
}

func vecScal(a []ristretto.Scalar, b ristretto.Scalar) ([]ristretto.Scalar, error) {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Mul(&a[i], &b)
	}

	return res, nil
}

// Given a scalar, return the sum of its powers from 0 to n-1
func vecPowerSum(a ristretto.Scalar, n uint64) ristretto.Scalar {

	res := ristretto.Scalar{}
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
// vecPowers(5, 3) = <5^0, 5^1, 5^2>
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
