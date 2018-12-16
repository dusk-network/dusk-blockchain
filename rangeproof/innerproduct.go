package rangeproof

import (
	"errors"
	"math/bits"

	"gitlab.dusk.network/dusk-core/dusk-go/rangeproof/vector"
	"gitlab.dusk.network/dusk-core/dusk-go/ristretto"
)

// XXX : eventually we will move this function out of the
// package and form an inner product package which will produce a reduced proof for the bullet proof

// Given two scalar arrays, construct the inner product
func innerProduct(a, b []ristretto.Scalar) (ristretto.Scalar, error) {

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

// IP sets up an inner product struct
type IP struct {
	G    []ristretto.Point
	H    []ristretto.Point
	a, b []ristretto.Scalar
	u    ristretto.Point
}

// IPProof represents an inner product proof
type IPProof struct {
	IP   IP
	L    []ristretto.Point
	R    []ristretto.Point
	a, b ristretto.Scalar
}

// NewIP creates a new Inner product struct
// making relevant checks on the given parameters
func NewIP(GVec, HVec []ristretto.Point, l, r []ristretto.Scalar, u ristretto.Point) (*IP, error) {

	// XXX: N.B. l and r are pointers
	// Alternative would be to not edit these directly in the Create proof method
	a := make([]ristretto.Scalar, len(l))
	copy(a, l)
	b := make([]ristretto.Scalar, len(r))
	copy(b, r)
	G := make([]ristretto.Point, len(GVec))
	copy(G, GVec)
	H := make([]ristretto.Point, len(HVec))
	copy(H, HVec)

	n := len(G)

	if n == 0 {
		return nil, errors.New("[IPProof]: size of n (NM) equals zero")
	}
	if len(H) != n {
		return nil, errors.New("[IPProof]: size of H does not equal n")
	}
	if len(a) != n {
		return nil, errors.New("[IPProof]: size of a does not equal n")
	}
	if len(b) != n {
		return nil, errors.New("[IPProof]: size of b does not equal n")
	}

	// XXX : When n is not a power of two, will the bulletproof struct pad it
	// or will the inner product proof struct?
	if !isPower2(uint32(n)) {
		return nil, errors.New("[IPProof]: size of n (NM) is not a power of 2")
	}

	return &IP{
		G: G,
		H: H,
		a: a,
		b: b,
		u: u,
	}, nil

}

// Create will generate a new proof
func (i *IP) Create() (*IPProof, error) {

	hs := hashCacher{[]byte{}}

	n := uint32(len(i.G))

	Lj := make([]ristretto.Point, 0) // XXX: Performance, will constantly allocate mem. N.B. size will be log(M*N)
	Rj := make([]ristretto.Point, 0)

	for n > 1 {

		// halve n
		n = n / 2

		aL, aR, err := vector.SplitScalars(i.a, n)
		bL, bR, err := vector.SplitScalars(i.b, n)
		GL, GR, err := vector.SplitPoints(i.G, n)
		HL, HR, err := vector.SplitPoints(i.H, n)

		cL, err := innerProduct(aL, bR)
		if err != nil {
			return nil, err
		}
		cR, err := innerProduct(aR, bL)
		if err != nil {
			return nil, err
		}

		// L = aL * GR + bR * HL + cL * u = e1 + e2 + e3

		e1, err := vector.Exp(aL, GR, int(n), 1)
		if err != nil {
			return nil, err
		}
		e2, err := vector.Exp(bR, HL, int(n), 1)
		if err != nil {
			return nil, err
		}
		var e3 ristretto.Point
		e3.ScalarMult(&i.u, &cL)

		var L ristretto.Point
		L.SetZero()
		L.Add(&e1, &e2)
		L.Add(&L, &e3)

		Lj = append(Lj, L)

		// R = aR * GL + bL * HR + cR * u = e4 + e5 + e6

		e4, err := vector.Exp(aR, GL, int(n), 1)
		if err != nil {
			return nil, err
		}
		e5, err := vector.Exp(bL, HR, int(n), 1)
		if err != nil {
			return nil, err
		}
		var e6 ristretto.Point
		e6.ScalarMult(&i.u, &cR)

		var R ristretto.Point
		R.SetZero()
		R.Add(&e4, &e5)
		R.Add(&R, &e6)
		Rj = append(Rj, R)

		hs.Append(L.Bytes(), R.Bytes())

		x := hs.Derive()
		var xinv ristretto.Scalar
		xinv.Inverse(&x)

		// aL = aL * x + aR *xinv = a1 + a2 - aprime
		// bL = bR * x + bL *xinv = b1 + b2 - bprime
		// GL = GL * xinv + GR * x = g1 + g2 - gprime
		// HL = HL * x + HR * xinv = h1 + h2 - hprime

		var a1, a2, b1, b2 ristretto.Scalar
		var g1, g2, h1, h2 ristretto.Point

		for i := uint32(0); i < n; i++ {

			a1.Mul(&aL[i], &x)
			a2.Mul(&aR[i], &xinv)
			aL[i].Add(&a1, &a2)

			b1.Mul(&bL[i], &xinv)
			b2.Mul(&bR[i], &x)
			bL[i].Add(&b1, &b2)

			g1.ScalarMult(&GL[i], &xinv)
			g2.ScalarMult(&GR[i], &x)
			GL[i].Add(&g1, &g2)

			h1.ScalarMult(&HL[i], &x)
			h2.ScalarMult(&HR[i], &xinv)
			HL[i].Add(&h1, &h2)
		}

		i.a = aL
		i.b = bL
		i.G = GL
		i.H = HL

	}

	return &IPProof{
		IP: *i,
		L:  Lj,
		R:  Rj,
		a:  i.a[len(i.a)-1],
		b:  i.b[len(i.b)-1],
	}, nil
}

// taken from rust implementation
func (i *IPProof) verifScalars() ([]ristretto.Scalar, []ristretto.Scalar, []ristretto.Scalar) {
	// generate scalars for verification

	if len(i.L) != len(i.R) {
		return nil, nil, nil
	}

	lgN := len(i.L)
	n := uint32(1 << uint(lgN))

	hs := hashCacher{[]byte{}}

	// 1. compute x's
	xChals := make([]ristretto.Scalar, 0, lgN)
	for k := range i.L {
		hs.Append(i.L[k].Bytes(), i.R[k].Bytes())
		xChals = append(xChals, hs.Derive())
	}

	// 2. compute inverse of x's
	invXChals := make([]ristretto.Scalar, 0, lgN)

	var invProd ristretto.Scalar // this will be the product of all of the inverses
	invProd.SetOne()

	for k := range xChals {

		var xinv ristretto.Scalar
		xinv.Inverse(&xChals[k])

		invProd.Mul(&invProd, &xinv)

		invXChals = append(invXChals, xinv)
	}

	// 3. compute x^2 and inv(x)^2
	chalSq := make([]ristretto.Scalar, 0, lgN)
	invChalSq := make([]ristretto.Scalar, 0, lgN)

	for k := range xChals {
		var sq ristretto.Scalar
		var invSq ristretto.Scalar

		sq.Square(&xChals[k])
		invSq.Square(&invXChals[k])

		chalSq = append(chalSq, sq)
		invChalSq = append(invChalSq, invSq)
	}

	// 4. compute s

	s := make([]ristretto.Scalar, 0, n)

	// push the inverse product
	s = append(s, invProd)

	for i := uint32(1); i < n; i++ {

		lgI := 32 - 1 - bits.LeadingZeros32(i)
		k := uint32(1 << uint(lgI))

		uLgISq := chalSq[(lgN-1)-lgI]

		var sRes ristretto.Scalar
		sRes.Mul(&s[i-k], &uLgISq)
		s = append(s, sRes)
	}

	return chalSq, invChalSq, s
}

func isPower2(n uint32) bool {
	return (n & (n - 1)) == 0
}
