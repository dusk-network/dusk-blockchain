package pedersen_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/rangeproof/pedersen"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

func TestPedersenScalar(t *testing.T) {
	ped := pedersen.New([]byte("random data"))

	s := ristretto.Scalar{}
	s.Rand()

	commitment := ped.CommitToScalar(s)

	assert.NotEqual(t, nil, commitment)

	fmt.Println(commitment.Value, commitment.BlindingFactor)
}

func TestPedersenScalarZero(t *testing.T) {
	ped := pedersen.New([]byte("random data"))

	s := ristretto.Scalar{}
	s.SetBigInt(big.NewInt(0))

	commitment := ped.CommitToScalar(s)

	assert.NotEqual(t, nil, commitment)

	// P = aG + bH
	aG := ristretto.Point{}
	aG.ScalarMultBase(&commitment.BlindingFactor)

	bH := ristretto.Point{}
	bH.ScalarMult(&ped.H, &s)

	aGbH := ristretto.Point{}
	aGbH.Add(&aG, &bH)

	assert.Equal(t, aGbH, commitment.Value)
}

func TestBlindingFactorSame(t *testing.T) {
	ped := pedersen.New([]byte("random data"))

	s := ristretto.Scalar{}
	s.SetBigInt(big.NewInt(100))

	cS := ped.CommitToScalar(s)

	expected := cS.BlindingFactor

	k := ristretto.Scalar{}
	k.SetBigInt(big.NewInt(100))

	cK := ped.CommitToScalar(k)

	cS.Add(cK)

	assert.Equal(t, expected, cS.BlindingFactor)
}
