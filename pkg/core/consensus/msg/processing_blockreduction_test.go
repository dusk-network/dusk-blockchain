package msg_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestVerifyBlockReduction(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	if err := ctx.Committee.AddMember(m.PubKey); err != nil {
		t.Fatal(err)
	}

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestBlockReductionWrongStep(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	if err := ctx.Committee.AddMember(m.PubKey); err != nil {
		t.Fatal(err)
	}

	// Change our step and verify the message (should fail with low priority error)
	ctx.BlockStep++
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("step check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockReductionNotInCommittee(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
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

func TestBlockReductionWrongBLSKey(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	if err := ctx.Committee.AddMember(m.PubKey); err != nil {
		t.Fatal(err)
	}

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

func TestBlockReductionWrongBLSSig(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 0, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	if err := ctx.Committee.AddMember(m.PubKey); err != nil {
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
