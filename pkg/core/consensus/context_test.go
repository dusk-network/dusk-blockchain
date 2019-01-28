package consensus

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestNewGeneratorContext(t *testing.T) {

	tau := rand.Uint64()

	keys, err := NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := NewContext(tau, 0, 150000, nil, protocol.TestNet, keys)
	assert.NotEqual(t, ctx, nil)
	assert.Equal(t, err, nil)

	assert.Equal(t, tau, ctx.Tau)
}

func TestReset(t *testing.T) {

	tau := rand.Uint64()

	keys, err := NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := NewContext(tau, 0, 150000, nil, protocol.TestNet, keys)

	// check consensus values were resetted
	assert.Nil(t, ctx.X, nil, nil)
	assert.Nil(t, ctx.Y, nil, nil)
	assert.Nil(t, ctx.Z, nil, nil)
	assert.Nil(t, ctx.M, nil, nil)
	assert.Nil(t, ctx.k, nil, nil)
	assert.Equal(t, uint64(0), ctx.Q)
	assert.Equal(t, uint64(0), ctx.d)
}

// Convenience function for provisioner tests
func provisionerContext() (*Context, error) {
	seed, _ := crypto.RandEntropy(32)
	keys, _ := NewRandKeys()
	totalWeight := uint64(500000)
	round := uint64(150000)
	ctx, err := NewContext(0, totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
