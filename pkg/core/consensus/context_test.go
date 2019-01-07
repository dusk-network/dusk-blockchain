package consensus

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {

	tau := rand.Uint64()

	keys, err := NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := NewContext(tau, keys)
	assert.NotEqual(t, ctx, nil)
	assert.Equal(t, err, nil)

	assert.Equal(t, tau, ctx.Tau)
}
