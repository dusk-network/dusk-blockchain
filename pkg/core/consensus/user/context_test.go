package user_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestNewGeneratorContext(t *testing.T) {

	tau := rand.Uint64()

	keys, err := user.NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := user.NewContext(tau, 0, 0, 150000, nil, protocol.TestNet, keys)
	assert.NotEqual(t, ctx, nil)
	assert.Equal(t, err, nil)

	assert.Equal(t, tau, ctx.Tau)
}

func TestReset(t *testing.T) {

	tau := rand.Uint64()

	keys, err := user.NewRandKeys()
	assert.Equal(t, err, nil)

	ctx, err := user.NewContext(tau, 0, 0, 150000, nil, protocol.TestNet, keys)

	// check consensus values were resetted
	assert.Equal(t, ctx.X, make([]byte, 32))
	assert.Equal(t, ctx.Y, make([]byte, 32))
	assert.Nil(t, ctx.Z, nil, nil)
	assert.Nil(t, ctx.M, nil, nil)
	assert.Equal(t, ctx.Q, make([]byte, 32))
}

// Convenience function for provisioner tests
func provisionerContext() (*user.Context, error) {
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	totalWeight := uint64(500000)
	round := uint64(150000)
	ctx, err := user.NewContext(0, 0, totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
