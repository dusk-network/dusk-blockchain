package rangeproof

import (
	"math/big"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/rangeproof/pedersen"

	"gitlab.dusk.network/dusk-core/dusk-go/rangeproof/vector"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/ristretto"
)

func TestInnerProduct(t *testing.T) {
	var one ristretto.Scalar
	one.SetBigInt(big.NewInt(1))
	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))
	var three ristretto.Scalar
	three.SetBigInt(big.NewInt(3))
	var four ristretto.Scalar
	four.SetBigInt(big.NewInt(4))
	var five ristretto.Scalar
	five.SetBigInt(big.NewInt(5))

	a := []ristretto.Scalar{one, two, three, four}
	b := []ristretto.Scalar{two, three, four, five}

	res, _ := innerProduct(a, b)

	var expected ristretto.Scalar
	expected.SetBigInt(big.NewInt(40))

	ok := expected.Equals(&res)

	assert.Equal(t, true, ok)
}

func TestInnerProductProof(t *testing.T) {

	const n = 32

	a := randomScalarArr(n)
	b := randomScalarArr(n)
	c, err := innerProduct(a, b)

	assert.Equal(t, nil, err)

	var yInv ristretto.Scalar
	yInv.Rand()

	var Q ristretto.Point
	Q.SetZero()

	HprimeFactors := vector.ScalarPowers(yInv, n)

	bPrime := make([]ristretto.Scalar, n)
	for i := range bPrime {
		bPrime[i].Mul(&b[i], &HprimeFactors[i])
	}

	aPrime := make([]ristretto.Scalar, n)
	copy(aPrime, a)

	// P = aPrime * G + bPrime * H + c * Q = k1 + k2 + k3
	var k1 ristretto.Point
	var k2 ristretto.Point
	var k3 ristretto.Point

	genData := []byte("dusk.BulletProof.vec1")
	ped := pedersen.New(genData)
	ped.BaseVector.Compute(n)

	genData = append(genData, uint8(1))

	ped2 := pedersen.New(genData)
	ped2.BaseVector.Compute(n)

	H := ped2.BaseVector.Bases
	G := ped.BaseVector.Bases

	k1, err = vector.Exp(aPrime, G, n, 1)
	k2, err = vector.Exp(bPrime, H, n, 1)
	k3.ScalarMult(&Q, &c)

	var P ristretto.Point
	P.SetZero()
	P.Add(&k1, &k2)
	P.Add(&P, &k3)

	ip, err := NewIP(G, H, aPrime, bPrime, Q)
	assert.Equal(t, nil, err)

	proof, err := ip.Create()
	assert.Equal(t, nil, err)
	uSq, uInvSq, s := proof.verifScalars()

	sInv := make([]ristretto.Scalar, len(s))
	copy(sInv, s)

	// reverse s
	for i, j := 0, len(sInv)-1; i < j; i, j = i+1, j-1 {
		sInv[i], sInv[j] = sInv[j], sInv[i]
	}

	aTimesS := vector.MulScalar(s, proof.a)
	bDivS := vector.MulScalar(sInv, proof.b)

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
	baseC.Mul(&proof.a, &proof.b)

	scalars = append(scalars, baseC)
	scalars = append(scalars, aTimesS...)
	scalars = append(scalars, bDivS...)
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
	assert.Equal(t, nil, err)

	t.Fail()

	assert.Equal(t, P.Bytes(), have.Bytes())
}

func randomScalarArr(n uint32) []ristretto.Scalar {
	res := make([]ristretto.Scalar, n)

	for i := range res {
		var rand ristretto.Scalar
		rand.Rand()
		res[i] = rand
	}
	return res
}
