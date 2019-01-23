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
	if !valid {
		t.Fatal("message was not valid")
	}

	// Votes should be equal
	assert.Equal(t, votes, retVotes)
}

func TestProcessMsgAgreement(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Create a binary agreement phase voting message and get their amount of votes
	votes, msg, err := newVoteAgreement(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Process the message
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid
	if !valid {
		t.Fatal("message was not valid")
	}

	// Votes should be equal
	assert.Equal(t, votes, retVotes)
}

// TODO: implement missing message types
