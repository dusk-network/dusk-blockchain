package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestAddDecoys(t *testing.T) {
	p := Proof{}
	assert.Equal(t, 0, len(p.pubKeysMatrix))

	for i := 0; i < 100; i++ {
		p.AddDecoy(generateDecoy(2))
		assert.Equal(t, i+1, len(p.pubKeysMatrix))
	}
}

func TestAddSecretKeys(t *testing.T) {

	// Add secret key, then check if corresponding pubKey gets added
	p := Proof{}

	decoy := generateDecoy(2)
	p.AddDecoy(decoy)
	assert.Equal(t, 1, len(p.pubKeysMatrix))

	realKeys := generateSks(2)
	for i := range realKeys {
		p.AddSecret(realKeys[i])
	}

	p.addSignerPubKey()

	assert.Equal(t, 2, len(p.pubKeysMatrix))

	// Check that the privKeys match the pubkeys
	var firstPubKey, secondPubKey ristretto.Point
	firstPubKey.ScalarMultBase(&realKeys[0])
	secondPubKey.ScalarMultBase(&realKeys[1])

	assert.True(t, p.pubKeysMatrix[1].keys[0] == firstPubKey)
	assert.True(t, p.pubKeysMatrix[1].keys[1] == secondPubKey)
}

func generateSks(n int) PrivKeys {
	p := PrivKeys{}

	for i := 0; i < n; i++ {
		var x ristretto.Scalar
		x.Rand()
		p.AddPrivateKey(x)
	}
	return p
}
