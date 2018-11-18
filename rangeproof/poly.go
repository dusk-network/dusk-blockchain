package rangeproof

import (
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// Polynomial construction
type polynomial struct {
	l0, l1, r0, r1 []ristretto.Scalar
	t0, t1, t2     ristretto.Scalar
}

func computePoly(aL, aR, sL, sR [N]ristretto.Scalar, y, z, v ristretto.Scalar) polynomial {

	// calculate l_0
	l0, _ := vecSubScal(aL[:], z)

	// calculate l_1
	l1 := sL

	// calculate r_0

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vecPowers(two, N)
	yN := vecPowers(y, N)

	var zsq ristretto.Scalar
	zsq.Square(&z)
	zsqTwoN, _ := vecScal(twoN, zsq)

	r0, _ := vecAddScal(aR[:], z)

	r0, _ = hadamard(r0, yN)

	r0, _ = vecAdd(r0, zsqTwoN)

	// calculate r_1
	r1, _ := hadamard(yN, sR[:])

	// calculate t0 // t_0 = <l_0, r_0>
	t0, _ := innerProduct(l0, r0)

	// calculate t1 // t_1 = <l_0, r_1> + <l_1, r_0>
	t1Left, _ := innerProduct(l1[:], r0[:])
	t1Right, _ := innerProduct(l0, r1)

	var t1 ristretto.Scalar
	t1.Add(&t1Left, &t1Right)

	// calculate t2 // t_2 = <l_1, r_1>
	t2, _ := innerProduct(l1[:], r1[:])

	return polynomial{
		l0: l0,
		l1: l1[:],
		r0: r0,
		r1: r1,
		t0: t0,
		t1: t1,
		t2: t2,
	}
}

// evalute the polynomial with coefficients t
// t = t_0 + t_1 * x + t_2 x^2
func (p *polynomial) eval(x ristretto.Scalar) ristretto.Scalar {

	var t1x ristretto.Scalar
	t1x.Mul(&x, &p.t1)

	var xsq ristretto.Scalar
	xsq.Square(&x)

	var t2xsq ristretto.Scalar
	t2xsq.Mul(&xsq, &p.t2)

	var t ristretto.Scalar
	t.Add(&t1x, &t2xsq)
	t.Add(&t, &p.t0)

	return t
}

// l = l_0 + l_1 * x
func (p *polynomial) computeL(x ristretto.Scalar) []ristretto.Scalar {

	lLeft := p.l0

	lRight, _ := vecScal(p.l1, x)

	l, _ := vecAdd(lLeft, lRight)

	return l
}

// r = r_0 + r_1 * x
func (p *polynomial) computeR(x ristretto.Scalar) []ristretto.Scalar {
	rLeft := p.r0

	rRight, _ := vecScal(p.r1, x)

	r, _ := vecAdd(rLeft, rRight)

	return r
}

// t_0 = z^2 * v + D(y,z)
func (p *polynomial) computeT0(y, z, v ristretto.Scalar) ristretto.Scalar {

	delta := computeDelta(y, z)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	var zsqv ristretto.Scalar
	zsqv.Mul(&zsq, &v)

	var t0 ristretto.Scalar
	t0.SetZero()

	t0.Add(&delta, &zsqv)

	return t0
}

// D(y,z) - This is the data shared by both prover and verifier
func computeDelta(y, z ristretto.Scalar) ristretto.Scalar {
	var res ristretto.Scalar
	res.SetZero()

	var one ristretto.Scalar
	one.SetOne()
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	oneDotYn := sumOfPowers(y, N)
	oneDot2n := sumOfPowers(two, N)

	var zsq ristretto.Scalar
	zsq.Square(&z)

	var zcu ristretto.Scalar
	zcu.Mul(&zsq, &z)

	var zMinusZsq ristretto.Scalar
	zMinusZsq.Sub(&z, &zsq)

	var zcuOneDot2n ristretto.Scalar
	zcuOneDot2n.Mul(&zcu, &oneDot2n)

	res.Mul(&zMinusZsq, &oneDotYn)
	res.Sub(&res, &zcuOneDot2n)

	return res
}
