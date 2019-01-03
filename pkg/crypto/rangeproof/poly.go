package rangeproof

import (
	"math/big"

	"github.com/pkg/errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/vector"
)

// Polynomial construction
type polynomial struct {
	l0, l1, r0, r1 []ristretto.Scalar
	t0, t1, t2     ristretto.Scalar
}

func computePoly(aL, aR, sL, sR []ristretto.Scalar, y, z ristretto.Scalar) (*polynomial, error) {

	// calculate l_0
	l0 := vector.SubScalar(aL, z)

	// calculate l_1
	l1 := sL

	// calculate r_0
	yNM := vector.ScalarPowers(y, uint32(N*M))

	zMTwoN := sumZMTwoN(z)

	r0 := vector.AddScalar(aR, z)

	r0, err := vector.Hadamard(r0, yNM)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - r0 (1)")
	}
	r0, err = vector.Add(r0, zMTwoN)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - r0 (2)")
	}

	// calculate r_1
	r1, err := vector.Hadamard(yNM, sR)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - r1")
	}

	// calculate t0 // t_0 = <l_0, r_0>
	t0, err := vector.InnerProduct(l0, r0)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - t0")
	}

	// calculate t1 // t_1 = <l_0, r_1> + <l_1, r_0>
	t1Left, err := vector.InnerProduct(l1, r0)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - t1Left")
	}
	t1Right, err := vector.InnerProduct(l0, r1)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - t1Right")
	}
	var t1 ristretto.Scalar
	t1.Add(&t1Left, &t1Right)

	// calculate t2 // t_2 = <l_1, r_1>
	t2, err := vector.InnerProduct(l1, r1)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputePoly] - t2")
	}
	return &polynomial{
		l0: l0,
		l1: l1[:],
		r0: r0,
		r1: r1,
		t0: t0,
		t1: t1,
		t2: t2,
	}, nil
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
func (p *polynomial) computeL(x ristretto.Scalar) ([]ristretto.Scalar, error) {

	lLeft := p.l0

	lRight := vector.MulScalar(p.l1, x)

	l, err := vector.Add(lLeft, lRight)
	if err != nil {
		return nil, errors.Wrap(err, "[ComputeL]")
	}
	return l, nil
}

// r = r_0 + r_1 * x
func (p *polynomial) computeR(x ristretto.Scalar) ([]ristretto.Scalar, error) {
	rLeft := p.r0

	rRight := vector.MulScalar(p.r1, x)

	r, err := vector.Add(rLeft, rRight)
	if err != nil {
		return nil, errors.Wrap(err, "[computeR]")
	}
	return r, nil
}

// t_0 = z^2 * v + D(y,z)
func (p *polynomial) computeT0(y, z ristretto.Scalar, v []ristretto.Scalar, n, m uint32) ristretto.Scalar {

	delta := computeDelta(y, z, n, uint32(m))

	var zSq ristretto.Scalar
	zSq.Square(&z)

	zM := vector.ScalarPowers(z, uint32(len(v)))
	zM = vector.MulScalar(zM, zSq)

	var sumZmV ristretto.Scalar
	sumZmV.SetZero()

	for i := range v {
		sumZmV.MulAdd(&zM[i], &v[i], &sumZmV)
	}

	var t0 ristretto.Scalar
	t0.SetZero()

	t0.Add(&delta, &sumZmV)

	return t0
}

// calculates sum( z^(1+j) * ( 0^(j-1)n || 2 ^n || 0^(m-j)n ) ) from j = 1 to j=M (71)
// implementation taken directly from java implementation.
// XXX: Look into ways to speed this up, and improve readability
// XXX: pass n and m as parameters
func sumZMTwoN(z ristretto.Scalar) []ristretto.Scalar {

	res := make([]ristretto.Scalar, N*M)

	zM := vector.ScalarPowers(z, uint32(M+3))

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	twoN := vector.ScalarPowers(two, N)

	for i := 0; i < M*N; i++ {
		res[i].SetZero()
		for j := 1; j <= M; j++ {
			if (i >= (j-1)*N) && (i < j*N) {
				res[i].MulAdd(&zM[j+1], &twoN[i-(j-1)*N], &res[i])
			}
		}

	}
	return res

	// Below is the old variation of the above code , for clarity on what the above is doing

	// res := make([]ristretto.Scalar, 0, N*M)

	// var zSq ristretto.Scalar
	// zSq.Square(&z)

	// zM := vector.ScalarPowers(z, uint32(M))
	// zM = vector.MulScalar(zM, z)

	// var two ristretto.Scalar
	// two.SetBigInt(big.NewInt(2))
	// twoN := vector.ScalarPowers(two, N)

	// for i := 0; i < M; i++ {
	// 	a := vector.MulScalar(twoN, zM[i])
	// 	res = append(res, a...)
	// }

	// return res

}

// D(y,z) - This is the data shared by both prover and verifier
// ported from rust impl
func computeDelta(y, z ristretto.Scalar, n, m uint32) ristretto.Scalar {

	var res ristretto.Scalar
	res.SetZero()

	sumY := vector.ScalarPowersSum(y, uint64(n*m))
	sumZ := vector.ScalarPowersSum(z, uint64(m))

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	sum2 := vector.ScalarPowersSum(two, uint64(n))
	var zsq ristretto.Scalar
	zsq.Square(&z)

	var zcu ristretto.Scalar
	zcu.Mul(&z, &zsq)

	var resA, resB ristretto.Scalar
	resA.Sub(&z, &zsq)

	resA.Mul(&resA, &sumY)

	resB.Mul(&sum2, &sumZ)
	resB.Mul(&resB, &zcu)

	res.Sub(&resA, &resB)

	return res
}
