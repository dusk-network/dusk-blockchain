package pedersen_test

import (
	"bytes"
	"math/big"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/pedersen"
)

func TestPedersenScalar(t *testing.T) {
	ped := pedersen.New([]byte("random data"))

	s := ristretto.Scalar{}
	s.Rand()

	commitment := ped.CommitToScalar(s)

	assert.NotEqual(t, nil, commitment)

}

func TestEncodeDecode(t *testing.T) {
	s := ristretto.Scalar{}
	s.Rand()

	c := pedersen.New([]byte("rand")).CommitToScalar(s)
	assert.True(t, c.Equals(c))

	buf := &bytes.Buffer{}
	err := c.Encode(buf)
	assert.Nil(t, err)

	var decC pedersen.Commitment
	err = decC.Decode(buf)
	assert.Nil(t, err)

	ok := decC.EqualValue(c)
	assert.True(t, ok)

}

func TestPedersenVector(t *testing.T) {
	ped := pedersen.New([]byte("some data"))
	// ped.BaseVector.Compute(4) // since values are not precomputed, we will compute two of them here
	var one ristretto.Scalar
	one.SetOne()

	var two ristretto.Scalar
	two.SetBigInt(big.NewInt(2))

	vec1 := []ristretto.Scalar{one, one}
	vec2 := []ristretto.Scalar{two, two}

	comm := ped.CommitToVectors(vec1, vec2)

	blind := comm.BlindingFactor

	H0 := ped.BlindPoint // blind
	H1 := ped.BaseVector.Bases[0]
	H2 := ped.BaseVector.Bases[1]

	ped = pedersen.New(append(ped.GenData, uint8(1)))

	ped.BaseVector.Compute(4) // since values are not precomputed, we will compute two of them here

	B0 := ped.BaseVector.Bases[0]
	B1 := ped.BaseVector.Bases[1]

	var H0blind ristretto.Point
	H0blind.ScalarMult(&H0, &blind)

	var H1one ristretto.Point
	H1one.ScalarMult(&H1, &one)

	var H2one ristretto.Point
	H2one.ScalarMult(&H2, &one)

	var B0two ristretto.Point
	B0two.ScalarMult(&B0, &two)

	var B1two ristretto.Point
	B1two.ScalarMult(&B1, &two)

	var expected ristretto.Point
	expected.Add(&H0blind, &H1one)
	expected.Add(&expected, &H2one)
	expected.Add(&expected, &B0two)
	expected.Add(&expected, &B1two)

	assert.Equal(t, expected.Bytes(), []byte(comm.Value.Bytes()))
}
