package rangeproof

import (
	"errors"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// XXX : eventually we will move this function out of the
// package and form an inner product package which will produce a reduced proof for the bullet proof

// Given two scalar arrays, construct the inner product
func innerProduct(a, b []ristretto.Scalar) (ristretto.Scalar, error) {

	res := ristretto.Scalar{}
	res.SetZero()

	if len(a) != len(b) {
		return res, errors.New("Length of a does not equal length of b")
	}

	for i := 0; i < len(a); i++ {
		res.MulAdd(&a[i], &b[i], &res)
	}

	return res, nil
}
