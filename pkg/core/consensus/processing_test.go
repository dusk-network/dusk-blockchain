package consensus_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

func TestFaultyMsgRound(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Change our round and verify the message (should fail)
	ctx.Round++
	retVotes, err2 := consensus.ProcessMsg(ctx, msg)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if retVotes != 0 && err2.Priority == prerror.Low {
		t.Fatal("wrong round did not get caught by check")
	}
}

func TestFaultyMsgStep(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Change our step and verify the message (should fail)
	ctx.Step++
	retVotes, err2 := consensus.ProcessMsg(ctx, msg)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if retVotes != 0 && err2.Priority == prerror.Low {
		t.Fatal("wrong step did not get caught by check")
	}
}

func TestFaultyMsgLastHeader(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Change our header hash and verify the message (should fail)
	ctx.LastHeader.Hash = make([]byte, 32)
	retVotes, err2 := consensus.ProcessMsg(ctx, msg)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if retVotes != 0 && err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgVersion(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Change our version and verify the message (should fail)
	ctx.Version = 20000
	retVotes, err2 := consensus.ProcessMsg(ctx, msg)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if retVotes != 0 && err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgSig(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Change their Ed25519 public key and verify the message (should fail)
	msg.PubKey = make([]byte, 32)
	retVotes, err2 := consensus.ProcessMsg(ctx, msg)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if retVotes != 0 && err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}
