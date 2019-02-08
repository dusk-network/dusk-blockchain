package msg_test

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

func TestFaultyMsgRound(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our round and verify the message (should fail)
	ctx.Round++
	err2 := msg.Process(ctx, m)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if err2.Priority == prerror.Low {
		t.Fatal("wrong round did not get caught by check")
	}
}

func TestFaultyMsgLastHeader(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our header hash and verify the message (should fail)
	ctx.LastHeader.Hash = make([]byte, 32)
	err2 := msg.Process(ctx, m)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgVersion(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our version and verify the message (should fail)
	ctx.Version = 20000
	err2 := msg.Process(ctx, m)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}

func TestFaultyMsgSig(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change their Ed25519 public key and verify the message (should fail)
	m.PubKey = make([]byte, 32)
	err2 := msg.Process(ctx, m)
	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}

	if err2.Priority == prerror.Low {
		t.Fatal("wrong version did not get caught by check")
	}
}

// Make a new consensus message
func newMessage(c *user.Context, blockHash []byte) (*payload.MsgConsensus, error) {
	// Make a context object
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(c.Tau, c.D, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	ctx.LastHeader = c.LastHeader
	sig, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, blockHash)
	if err != nil {
		return nil, err
	}

	// Create a random payload
	pl, err := consensusmsg.NewReduction(blockHash, sig, keys.BLSPubKey.Marshal())

	// Complete message and return it
	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, c.LastHeader.Hash, ctx.Step,
		sigEd, []byte(*keys.EdPubKey), pl)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
