package generation_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestSigSetGeneration(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = hash
	var voteSet []*consensusmsg.Vote
	for i := 0; i < 10; i++ {
		// Create random vote
		blsPk, _ := crypto.RandEntropy(129)
		sig, _ := crypto.RandEntropy(33)
		consensusmsg.NewVote(hash, blsPk, sig, 1)
	}

	ctx.SigSetVotes = voteSet
	ctx.Weight = 200

	// Run signature set generation function
	if err := generation.SignatureSet(ctx); err != nil {
		t.Fatal(err)
	}
}
