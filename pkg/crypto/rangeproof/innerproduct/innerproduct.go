package innerproduct

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/bits"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/fiatshamir"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/vector"
)

// This is a reference of the innerProduct implementation at rust

// Proof represents an innner product proof struct
type Proof struct {
	L, R []ristretto.Point
	A, B ristretto.Scalar // a and b are capitalised so that they are exported, in paper it is `a``b`
}

// Generate generates an inner product proof or an error
// if proof cannot be constucted
func Generate(GVec, HVec []ristretto.Point, aVec, bVec, HprimeFactors []ristretto.Scalar, Q ristretto.Point) (*Proof, error) {
	n := uint32(len(GVec))

	// XXX : When n is not a power of two, will the bulletproof struct pad it
	// or will the inner product proof struct?
	if !isPower2(uint32(n)) {
		return nil, errors.New("[IPProof]: size of n (NM) is not a power of 2")
	}

	a := make([]ristretto.Scalar, len(aVec))
	copy(a, aVec)
	b := make([]ristretto.Scalar, len(bVec))
	copy(b, bVec)
	G := make([]ristretto.Point, len(GVec))
	copy(G, GVec)
	H := make([]ristretto.Point, len(HVec))
	copy(H, HVec)

	hs := fiatshamir.HashCacher{[]byte{}}

	lgN := bits.TrailingZeros(nextPow2(uint(n)))

	Lj := make([]ristretto.Point, 0, lgN)
	Rj := make([]ristretto.Point, 0, lgN)

	if n != 1 {
		n = n / 2

		aL, aR, err := vector.SplitScalars(a, n)
		bL, bR, err := vector.SplitScalars(b, n)
		GL, GR, err := vector.SplitPoints(G, n)
		HL, HR, err := vector.SplitPoints(H, n)

		cL, err := vector.InnerProduct(aL, bR)
		if err != nil {

		}
		cR, err := vector.InnerProduct(aR, bL)
		if err != nil {
			return nil, err
		}

		// L = aL * GR + bR * HL * HPrime[0..n] + cL * Q = e1 + e2 + e3

		e1, err := vector.Exp(aL, GR, int(n), 1)
		if err != nil {
			return nil, err

		}

		bRYi := make([]ristretto.Scalar, len(bR))
		copy(bRYi, bR)

		for i := range bRYi {
			bRYi[i].Mul(&bRYi[i], &HprimeFactors[i])
		}

		e2, err := vector.Exp(bRYi, HL, int(n), 1)
		if err != nil {
			return nil, err
		}

		var e3 ristretto.Point
		e3.ScalarMult(&Q, &cL)

		var L ristretto.Point
		L.SetZero()
		L.Add(&e1, &e2)
		L.Add(&L, &e3)

		Lj = append(Lj, L)

		// R = aR * GL + bL * HR * HPrimeFactors[n .. 2n] + cR * Q = e4 + e5 + e6

		e4, err := vector.Exp(aR, GL, int(n), 1)
		if err != nil {
			return nil, err
		}

		bLYi := make([]ristretto.Scalar, len(bL))
		copy(bLYi, bL)

		for i := range bLYi {
			bLYi[i].Mul(&bLYi[i], &HprimeFactors[uint32(i)+n])
		}

		e5, err := vector.Exp(bLYi, HR, int(n), 1)
		if err != nil {
			return nil, err
		}

		var e6 ristretto.Point
		e6.ScalarMult(&Q, &cR)

		var R ristretto.Point
		R.SetZero()
		R.Add(&e4, &e5)
		R.Add(&R, &e6)
		Rj = append(Rj, R)

		hs.Append(L.Bytes(), R.Bytes())

		u := hs.Derive()
		var uinv ristretto.Scalar
		uinv.Inverse(&u)

		var a1, a2, b1, b2, h1a, h2a ristretto.Scalar
		var g1, g2, h1, h2 ristretto.Point

		for i := uint32(0); i < n; i++ {

			a1.Mul(&aL[i], &u)
			a2.Mul(&aR[i], &uinv)
			aL[i].Add(&a1, &a2)

			b1.Mul(&bL[i], &uinv)
			b2.Mul(&bR[i], &u)
			bL[i].Add(&b1, &b2)

			g1.ScalarMult(&GL[i], &uinv)
			g2.ScalarMult(&GR[i], &u)
			GL[i].Add(&g1, &g2)

			h1a.Mul(&HprimeFactors[i], &u)
			h1.ScalarMult(&HL[i], &h1a)
			h2a.Mul(&HprimeFactors[i+n], &uinv)
			h2.ScalarMult(&HR[i], &h2a)
			HL[i].Add(&h1, &h2)
		}

		a = aL
		b = bL
		G = GL
		H = HL
	}

	for n > 1 {

		n = n / 2

		aL, aR, err := vector.SplitScalars(a, n)
		bL, bR, err := vector.SplitScalars(b, n)
		GL, GR, err := vector.SplitPoints(G, n)
		HL, HR, err := vector.SplitPoints(H, n)

		cL, err := vector.InnerProduct(aL, bR)
		if err != nil {
			return nil, err
		}
		cR, err := vector.InnerProduct(aR, bL)
		if err != nil {
			return nil, err
		}

		// L = aL * GR + bR * HL + cL * Q = e1 + e2 + e3

		e1, err := vector.Exp(aL, GR, int(n), 1)
		if err != nil {
			return nil, err
		}
		e2, err := vector.Exp(bR, HL, int(n), 1)
		if err != nil {
			return nil, err
		}
		var e3 ristretto.Point
		e3.ScalarMult(&Q, &cL)

		var L ristretto.Point
		L.SetZero()
		L.Add(&e1, &e2)
		L.Add(&L, &e3)

		Lj = append(Lj, L)

		// R = aR * GL + bL * HR + cR * Q = e4 + e5 + e6

		e4, err := vector.Exp(aR, GL, int(n), 1)
		if err != nil {
			return nil, err
		}
		e5, err := vector.Exp(bL, HR, int(n), 1)
		if err != nil {
			return nil, err
		}
		var e6 ristretto.Point
		e6.ScalarMult(&Q, &cR)

		var R ristretto.Point
		R.SetZero()
		R.Add(&e4, &e5)
		R.Add(&R, &e6)
		Rj = append(Rj, R)

		hs.Append(L.Bytes(), R.Bytes())

		u := hs.Derive()
		var uinv ristretto.Scalar
		uinv.Inverse(&u)

		// aL = aL * u + aR *uinv = a1 + a2 - aprime
		// bL = bR * u + bL *uinv = b1 + b2 - bprime
		// GL = GL * uinv + GR * u = g1 + g2 - gprime
		// HL = HL * u + HR * uinv = h1 + h2 - hprime

		var a1, a2, b1, b2 ristretto.Scalar
		var g1, g2, h1, h2 ristretto.Point

		for i := uint32(0); i < n; i++ {

			a1.Mul(&aL[i], &u)
			a2.Mul(&aR[i], &uinv)
			aL[i].Add(&a1, &a2)

			b1.Mul(&bL[i], &uinv)
			b2.Mul(&bR[i], &u)
			bL[i].Add(&b1, &b2)

			g1.ScalarMult(&GL[i], &uinv)
			g2.ScalarMult(&GR[i], &u)
			GL[i].Add(&g1, &g2)

			h1.ScalarMult(&HL[i], &u)
			h2.ScalarMult(&HR[i], &uinv)
			HL[i].Add(&h1, &h2)
		}

		a = aL
		b = bL
		G = GL
		H = HL
	}

	return &Proof{
		L: Lj,
		R: Rj,
		A: a[len(a)-1],
		B: b[len(b)-1],
	}, nil
}

// VerifScalars generates the challenge squared, the inverse challenge squared
// and s for a given inner product proof
func (proof *Proof) VerifScalars() ([]ristretto.Scalar, []ristretto.Scalar, []ristretto.Scalar) {
	// generate scalars for verification

	if len(proof.L) != len(proof.R) {
		return nil, nil, nil
	}

	lgN := len(proof.L)
	n := uint32(1 << uint(lgN))

	hs := fiatshamir.HashCacher{Cache: []byte{}}

	// 1. compute x's
	xChals := make([]ristretto.Scalar, 0, lgN)
	for k := range proof.L {
		hs.Append(proof.L[k].Bytes(), proof.R[k].Bytes())
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

// Verify is used for unit tests and verifies that a given proof evaluates to the point P
func (proof *Proof) Verify(G, H, L, R []ristretto.Point, HprimeFactor []ristretto.Scalar, Q, P ristretto.Point, n int) bool {
	uSq, uInvSq, s := proof.VerifScalars()

	sInv := make([]ristretto.Scalar, len(s))
	copy(sInv, s)

	// reverse s
	for i, j := 0, len(sInv)-1; i < j; i, j = i+1, j-1 {
		sInv[i], sInv[j] = sInv[j], sInv[i]
	}

	aTimesS := vector.MulScalar(s, proof.A)
	hTimesbDivS := vector.MulScalar(sInv, proof.B)
	for i, bDivS := range hTimesbDivS {
		hTimesbDivS[i].Mul(&bDivS, &HprimeFactor[i])
	}

	negUSq := make([]ristretto.Scalar, len(uSq))
	for i := range negUSq {
		negUSq[i].Neg(&uSq[i])
	}

	negUInvSq := make([]ristretto.Scalar, len(uInvSq))
	for i := range negUInvSq {
		negUInvSq[i].Neg(&uInvSq[i])
	}

	// Scalars
	scalars := make([]ristretto.Scalar, 0)

	var baseC ristretto.Scalar
	baseC.Mul(&proof.A, &proof.B)

	scalars = append(scalars, baseC)
	scalars = append(scalars, aTimesS...)
	scalars = append(scalars, hTimesbDivS...)
	scalars = append(scalars, negUSq...)
	scalars = append(scalars, negUInvSq...)

	// Points
	points := make([]ristretto.Point, 0)
	points = append(points, Q)
	points = append(points, G...)
	points = append(points, H...)
	points = append(points, proof.L...)
	points = append(points, proof.R...)

	have, err := vector.Exp(scalars, points, n, 1)
	if err != nil {
		return false
	}
	return have.Equals(&P)
}

func (p *Proof) Encode(w io.Writer) error {

	err := binary.Write(w, binary.BigEndian, p.A.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, p.B.Bytes())
	if err != nil {
		return err
	}
	lenL := uint32(len(p.L))

	for i := uint32(0); i < lenL; i++ {
		err = binary.Write(w, binary.BigEndian, p.L[i].Bytes())
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, p.R[i].Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Proof) Decode(r io.Reader) error {
	if p == nil {
		return errors.New("struct is nil")
	}

	var ABytes, BBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &ABytes)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &BBytes)
	if err != nil {
		return err
	}
	p.A.SetBytes(&ABytes)
	p.B.SetBytes(&BBytes)

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(r)
	if err != nil {
		return err
	}
	numBytes := len(buf.Bytes())
	if numBytes%32 != 0 {
		return errors.New("proof was not formatted correctly")
	}
	lenL := uint32(numBytes / 64)

	p.L = make([]ristretto.Point, lenL)
	p.R = make([]ristretto.Point, lenL)

	for i := uint32(0); i < lenL; i++ {
		var LBytes, RBytes [32]byte
		err = binary.Read(buf, binary.BigEndian, &LBytes)
		if err != nil {
			return err
		}
		err = binary.Read(buf, binary.BigEndian, &RBytes)
		if err != nil {
			return err
		}
		p.L[i].SetBytes(&LBytes)
		p.R[i].SetBytes(&RBytes)
	}

	return nil
}

func (p *Proof) Equals(other Proof) bool {
	ok := p.A.Equals(&other.A)
	if !ok {
		return ok
	}

	ok = p.B.Equals(&other.B)
	if !ok {
		return ok
	}

	for i := range p.L {
		ok := p.L[i].Equals(&other.L[i])
		if !ok {
			return ok
		}

		ok = p.R[i].Equals(&other.R[i])
		if !ok {
			return ok
		}
	}

	return ok
}

func nextPow2(n uint) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n
}

func isPower2(n uint32) bool {
	return (n & (n - 1)) == 0
}

func DiffNextPow2(n uint32) uint32 {
	pow2 := nextPow2(uint(n))
	padAmount := uint32(pow2) - n + 1
	return padAmount
}
