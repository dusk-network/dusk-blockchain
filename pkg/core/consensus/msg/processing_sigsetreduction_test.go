package msg_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

func TestVerifySigSetVote(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestSigSetVoteDeviatingBlock(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our block hash
	otherBlock, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = otherBlock

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("unknown block check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetVoteWrongStep(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our step and verify the message (should fail with low priority error)
	ctx.Step++
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetVoteNotInCommittee(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetVoteWrongBLSKey(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash
	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Clear out our bls key mapping
	ctx.NodeBLS = make(map[string][]byte)

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetVoteWrongBLSSig(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message with wrong signature
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash
	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Verify the message (should fail with low priority error)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}
