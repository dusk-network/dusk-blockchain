package msg_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestVerifySigSetCandidate(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 0, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set and message
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x04, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.SigSetStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestSigSetCandidateDeviatingBlock(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 0, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Create vote set
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x04, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Increment step counter, and set up committee
	ctx.SigSetStep++
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.SigSetStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message (should fail with low priority error)
	ctx.WinningBlockHash = make([]byte, 32)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("deviating block check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}
