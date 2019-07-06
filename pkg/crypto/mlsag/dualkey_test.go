package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestDualKey(t *testing.T) {

	dk := generateRandDualKeyProof(20)

	sig, keyImage, err := dk.Prove()
	sig.Verify([]ristretto.Point{keyImage})
	assert.Nil(t, err)
}

func TestSubCommToZero(t *testing.T) {
	dk := generateRandDualKeyProof(20)

	var oldPoints []ristretto.Point

	// Save all of the old keys
	for i := range dk.pubKeysMatrix {
		oldPoint := dk.pubKeysMatrix[i].keys[1]
		oldPoints = append(oldPoints, oldPoint)
	}

	// Generate random point and subtract from commitment to zero column
	var x ristretto.Point
	x.Rand()
	dk.SubCommToZero(x)

	for i := range dk.pubKeysMatrix {
		var expectedPoint ristretto.Point

		// Fetch value of the second key
		// before subtracting x
		newPoint := dk.pubKeysMatrix[i].keys[1]

		// Calculate expected point
		expectedPoint.Sub(&oldPoints[i], &x)

		assert.True(t, expectedPoint.Equals(&newPoint))
	}
}

func generateRandDualKeyProof(numUsers int) *DualKey {
	proof := NewDualKey()

	numDecoys := numUsers - 1
	numKeys := 2

	// Generate and add decoys to proof
	matrixPubKeys := generateDecoys(numDecoys, numKeys)
	for i := 0; i < len(matrixPubKeys); i++ {
		pubKeys := matrixPubKeys[i]
		proof.AddDecoy(pubKeys)
	}

	// Generate and add private keys to proof
	var primaryKey, commToZero ristretto.Scalar
	primaryKey.Rand()
	commToZero.Rand()
	proof.SetPrimaryKey(primaryKey)
	proof.SetCommToZero(commToZero)

	proof.msg = []byte("hello world")

	return proof
}
