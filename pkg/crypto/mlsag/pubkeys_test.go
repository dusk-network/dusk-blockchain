package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestAddPubKey(t *testing.T) {

	var keys PubKeys
	keys.decoy = true
	assert.Equal(t, 0, keys.Len())

	for i := 0; i < 100; i++ {
		var p ristretto.Point
		p.Rand()

		keys.AddPubKey(p)
		assert.Equal(t, i+1, keys.Len())

		assert.True(t, keys.decoy)
	}
}
