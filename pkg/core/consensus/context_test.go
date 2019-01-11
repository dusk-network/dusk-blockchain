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

func TestReset(t *testing.T) {

	tau := rand.Uint64()

	keys, err := NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := NewContext(tau, keys)

	// check consensus values were resetted
	assert.Nil(t, ctx.X, nil, nil)
	assert.Nil(t, ctx.Y, nil, nil)
	assert.Nil(t, ctx.Z, nil, nil)
	assert.Nil(t, ctx.M, nil, nil)
	assert.Nil(t, ctx.k, nil, nil)
	assert.Equal(t, uint64(0), ctx.Q)
	assert.Equal(t, uint64(0), ctx.d)
}
