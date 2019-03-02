package generation_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

//XXX: Add fixed test input vectors to outputs
func TestBlockGeneration(t *testing.T) {
	ctx, err := user.NewContext(20, 5000, 0, 150000, nil, protocol.TestNet, randtestKeys(t))
	assert.Nil(t, err)
	ctx.K.Rand()

	err = generation.Block(ctx)
	assert.Nil(t, err)
}

// helper to generate random consensus keys
func randtestKeys(t *testing.T) *user.Keys {
	keys, err := user.NewRandKeys()
	if err != nil {
		t.FailNow()
	}
	return keys
}
