package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

func TestProcessMsgReduction(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a reduction phase voting message and get their amount of votes
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	votes, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Process the message
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Message should be valid
	if retVotes == 0 || !valid {
		t.Fatal("message was not valid")
	}

	// Votes should be equal
	assert.Equal(t, votes, retVotes)
}

// TODO: implement missing message types

func TestFaultyMsgRound(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

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
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	if retVotes != 0 && valid {
		t.Fatal("wrong round did not get caught by check")
	}
}

func TestFaultyMsgStep(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

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
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	if retVotes != 0 && valid {
		t.Fatal("wrong step did not get caught by check")
	}
}

func TestFaultyMsgLastHeader(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

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
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	if retVotes != 0 && valid {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgVersion(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

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
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	if retVotes != 0 && valid {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgSig(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

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
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	if retVotes != 0 && valid {
		t.Fatal("wrong version did not get caught by check")
	}
}
