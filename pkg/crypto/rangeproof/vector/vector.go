package vector

import (
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

// Add adds two scalar slices a and b,
// returning the resulting slice and an error
// if a and b were different sizes
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

// AddScalar takes a scalar slice a and a scalar, b
// then adds b to every element in a
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

// Neg Negates a vector a
func Neg(a []ristretto.Scalar) []ristretto.Scalar {
	if len(a) == 0 {
		return a
	}

	res := make([]ristretto.Scalar, len(a))

	for i := range a {
		res[i].Neg(&a[i])
	}

	return res
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

// MulScalar take a scalar b, and a vector a
// then multiplies every element in the scalar vector by b
func MulScalar(a []ristretto.Scalar, b ristretto.Scalar) []ristretto.Scalar {

	res := make([]ristretto.Scalar, len(a))

	for i := 0; i < len(a); i++ {
		res[i].Mul(&a[i], &b)
	}

	return res
}

// InnerProduct takes two scalar arrays and constructs the inner product
func InnerProduct(a, b []ristretto.Scalar) (ristretto.Scalar, error) {

	res := ristretto.Scalar{}
	res.SetZero()

	if len(a) != len(b) {
		return res, errors.New("[Inner Product]:Length of a does not equal length of b")
	}

	for i := 0; i < len(a); i++ {
		res.MulAdd(&a[i], &b[i], &res)
	}

	return res, nil
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

// ScalarPowers constructs a vector of powers
// vecPowers(5, 3) = <5^0, 5^1, 5^2>
func ScalarPowers(a ristretto.Scalar, n uint32) []ristretto.Scalar {

	res := make([]ristretto.Scalar, n)

	if n == 0 {
		return res
	}

	if a.IsNonZeroI() == 0 {
		return res
	}

	var k ristretto.Scalar
	k.SetOne()
	res[0] = k

	if n == 1 {
		return res
	}
	res[1] = a

	for i := uint32(2); i < n; i++ {
		res[i].Mul(&res[i-1], &a)
	}

	return res
}

// ScalarPowersSum constructs the Scalar power and then sums up each value
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

// Hadamard takes two scalar arrays and construct the Hadamard product
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

// FromScalar takes a scalar,a, scaToVec will return a slice of size n, with all elements equal to a
func FromScalar(a ristretto.Scalar, n uint32) []ristretto.Scalar {
	res := make([]ristretto.Scalar, n)

	for i := uint32(0); i < n; i++ {
		res[i] = a
	}

	return res
}

// SplitPoints will split a slice x using n
// Result will be two slices a and b where a = [0,n) and b = [n, len(x))
func SplitPoints(x []ristretto.Point, n uint32) ([]ristretto.Point, []ristretto.Point, error) {
	if len(x) <= 0 {
		return nil, nil, errors.New("Original vector has length of zero")
	}
	if n >= uint32(len(x)) {
		return nil, nil, errors.New("n is larger than the size of the slice x")
	}
	return x[:n], x[n:], nil
}

// SplitScalars will split a slice x using n
// Result will be two slices a and b where a = [0,n) and b = [n, len(x))
// Method same as above
// XXX: use unit test to make sure they output same sizes
func SplitScalars(x []ristretto.Scalar, n uint32) ([]ristretto.Scalar, []ristretto.Scalar, error) {
	if len(x) <= 0 {
		return nil, nil, errors.New("Original vector has length of zero")
	}
	if n >= uint32(len(x)) {
		return nil, nil, errors.New("n is larger than the size of the slice x")
	}
	return x[:n], x[n:], nil
}
