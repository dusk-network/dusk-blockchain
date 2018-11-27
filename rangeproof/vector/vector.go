package vector

import (
	"errors"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func Add(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}

	return res, nil
}
func AddScalar(a []ristretto.Scalar, b ristretto.Scalar) []ristretto.Scalar {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b)
	}

	return res
}

// Sub subtracts a vector a from a vector b
func Sub(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Sub(&a[i], &b[i])
	}

	return res, nil
}

// SubScalar Subtracts a scalars value b, from every element in the slice a
func SubScalar(a []ristretto.Scalar, b ristretto.Scalar) []ristretto.Scalar {

	if b.IsNonZeroI() == 0 {
		return a
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Sub(&a[i], &b)
	}

	return res
}

func MulScalar(a []ristretto.Scalar, b ristretto.Scalar) []ristretto.Scalar {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Mul(&a[i], &b)
	}

	return res
}

// Exp exponentiates and sums a vector a to b, creating a commitment
func Exp(a []ristretto.Scalar, b []ristretto.Point, N, M int) (ristretto.Point, error) {
	result := ristretto.Point{} // defaults to zero
	result.SetZero()

	if len(a) != len(b) {
		return result, errors.New("length of slice of scalars a does not equal length of slices of points b")
	}

	if len(a) < N*M {
		return result, errors.New("length of scalar a is not less than N*M")
	}

	for i := range b {

		scalar := a[i]
		point := b[i]

		var sum ristretto.Point
		sum.ScalarMult(&point, &scalar)

		result.Add(&result, &sum)

	}

	return result, nil
}

// Given a scalar, construct a vector of powers
// vecPowers(5, 3) = <5^0, 5^1, 5^2>
func ScalarPowers(a ristretto.Scalar, n uint8) []ristretto.Scalar {

	res := make([]ristretto.Scalar, n)

	if n == 0 {
		return res
	}

	// identity
	var k ristretto.Scalar
	k.SetOne() // Identity point
	res[0] = k

	if n == 1 {
		return res
	}
	res[1] = a

	for i := uint8(2); i < n; i++ {
		res[i].Mul(&res[i-1], &a)
	}

	return res
}

// Given two scalar arrays, construct the Hadamard product
func Hadamard(a, b []ristretto.Scalar) ([]ristretto.Scalar, error) {

	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal length of b")
	}

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Mul(&a[i], &b[i])
	}
	return res, nil
}

// Given two curvepoint arrays, construct the Hadamard product
func Hadamard2(a, b []ristretto.Point) ([]ristretto.Point, error) {

	if len(a) != len(b) {
		return nil, errors.New("Length of a does not equal length of b")
	}

	res := make([]ristretto.Point, len(a))
	for i := 0; i < len(a); i++ {
		res[i].Add(&a[i], &b[i])
	}

	return res, nil
}

// given a scalar,a, scaToVec will return a slice of size n, with all elements equal to a
func FromScalar(a ristretto.Scalar, n uint8) []ristretto.Scalar {
	res := make([]ristretto.Scalar, n)

	for i := uint8(0); i < n; i++ {
		res[i] = a
	}

	return res
}

func ScalarPowersSum(a ristretto.Scalar, n uint64) ristretto.Scalar {

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
