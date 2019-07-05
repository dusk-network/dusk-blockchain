package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestAddPrivKey(t *testing.T) {

	var keys PrivKeys
	assert.Equal(t, 0, keys.Len())

	for i := 0; i < 100; i++ {
		var s ristretto.Scalar
		s.Rand()
		keys.AddPrivateKey(s)
		assert.Equal(t, i+1, keys.Len())
	}

}
