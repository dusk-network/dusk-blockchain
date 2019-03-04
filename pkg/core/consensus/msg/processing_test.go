package msg_test

import (
	"encoding/hex"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our round and verify the message (should fail)
	ctx.Round++
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("round check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our header hash and verify the message (should fail)
	ctx.LastHeader.Hash = make([]byte, 32)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("chain check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change our version and verify the message (should fail)
	ctx.Version = 20000
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("version check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Change their Ed25519 public key and verify the message (should fail)
	m.PubKey = make([]byte, 32)
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("signature check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestFutureMsgRoundBlock(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Decrement our round and verify the message (should go into ctx.BlockQueue)
	ctx.Round--
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal("round check did not work")
	}

	_, ok := ctx.BlockQueue.Load(ctx.Round + 1)
	assert.True(t, ok)
}

func TestFutureMsgStepBlock(t *testing.T) {
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

	m, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m.PubKey)

	// Decrement our step and verify the message (should go into ctx.todo)
	ctx.BlockStep--
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal("round check did not work")
	}

	_, ok := ctx.BlockQueue.Load(ctx.Round)
	assert.True(t, ok)
}

func TestProcessingBlockQueue(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create dummy messages to store (1 step ahead)
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockStep++
	m1, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	m3, err := newMessage(ctx, emptyBlock.Header.Hash, 0x02, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m1.PubKey)
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m2.PubKey)
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m3.PubKey)

	// Decrement our step and verify the messages (should go into ctx.todo)
	ctx.BlockStep--
	err2 := msg.Process(ctx, m1)
	if err2 != nil {
		t.Fatal(err)
	}

	err2 = msg.Process(ctx, m2)
	if err2 != nil {
		t.Fatal(err)
	}

	err2 = msg.Process(ctx, m3)
	if err2 != nil {
		t.Fatal(err)
	}

	_, ok := ctx.BlockQueue.Load(ctx.Round)
	assert.True(t, ok)

	// Now increment the step again and verify another message
	ctx.BlockStep++
	if err2 := msg.ProcessBlockQueue(ctx); err2 != nil {
		t.Fatal(err)
	}

	// Messages should have been verified and deleted
	arr := ctx.BlockQueue.Get(ctx.Round, ctx.BlockStep)
	assert.Empty(t, arr)
}

func TestFutureMsgRoundSigSet(t *testing.T) {
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

	// Decrement our round and verify the message (should go into ctx.BlockQueue)
	ctx.Round--
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal("round check did not work")
	}

	_, ok := ctx.SigSetQueue.Load(ctx.Round + 1)
	assert.True(t, ok)
}

func TestFutureMsgStepSigSet(t *testing.T) {
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

	// Decrement our step and verify the message (should go into ctx.todo)
	ctx.SigSetStep--
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal("round check did not work")
	}

	_, ok := ctx.SigSetQueue.Load(ctx.Round)
	assert.True(t, ok)
}

func TestProcessingSigSetQueue(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create dummy messages to store (1 step ahead)
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetStep++
	m1, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	m3, err := newMessage(ctx, emptyBlock.Header.Hash, 0x05, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add them to our committee
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m1.PubKey)
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m2.PubKey)
	ctx.CurrentCommittee = append(ctx.CurrentCommittee, m3.PubKey)

	// Decrement our step and verify the messages (should go into ctx.todo)
	ctx.SigSetStep--
	err2 := msg.Process(ctx, m1)
	if err2 != nil {
		t.Fatal(err)
	}

	err2 = msg.Process(ctx, m2)
	if err2 != nil {
		t.Fatal(err)
	}

	err2 = msg.Process(ctx, m3)
	if err2 != nil {
		t.Fatal(err)
	}

	_, ok := ctx.SigSetQueue.Load(ctx.Round)
	assert.True(t, ok)

	// Now increment the step again and verify another message
	ctx.SigSetStep++
	if err2 := msg.ProcessSigSetQueue(ctx); err2 != nil {
		t.Fatal(err)
	}

	// Messages should have been verified and deleted
	arr := ctx.SigSetQueue.Get(ctx.Round, ctx.SigSetStep)
	assert.Empty(t, arr)
}

// Make a new consensus message of specified type
func newMessage(c *user.Context, blockHash []byte, id uint8,
	voteSet []*consensusmsg.Vote, spoofSig bool) (*payload.MsgConsensus, error) {
	// Make a context object
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(c.Tau, 1000, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	pkEd := hex.EncodeToString(keys.EdPubKeyBytes())
	pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
	c.NodeWeights[pkEd] = 500
	c.W += 500
	c.NodeBLS[pkBLS] = keys.EdPubKeyBytes()
	ctx.LastHeader = c.LastHeader
	ctx.BlockStep = c.BlockStep
	ctx.SigSetStep = c.SigSetStep
	ctx.BlockHash = blockHash
	ctx.Weight = 500
	ctx.K.Rand()

	// Create a payload
	// byte32, err := crypto.RandEntropy(32)
	// if err != nil {
	// 	return nil, err
	// }

	seed, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, ctx.Seed)
	if err != nil {
		return nil, err
	}

	ctx.Seed = seed

	var pl consensusmsg.Msg
	var step uint32
	switch consensusmsg.ID(id) {
	case consensusmsg.CandidateScoreID:
		dScalar := zkproof.Uint64ToScalar(ctx.D)

		m := zkproof.CalculateM(ctx.K)
		x := zkproof.CalculateX(dScalar, m)
		c.PubList.AddBid()

		seedScalar := ristretto.Scalar{}
		seedScalar.Derive(ctx.Seed)

		pL := make([]ristretto.Scalar, 0)
		proof, q, z, pubList := zkproof.Prove(dScalar, ctx.K, seedScalar, pL)

		pl, err = consensusmsg.NewCandidateScore(q, proof, z, blockHash, ctx.Seed, pubList)
		if err != nil {
			return nil, err
		}

		step = ctx.BlockStep
	case consensusmsg.CandidateID:
		blk, err := block.NewEmptyBlock(ctx.LastHeader)
		if err != nil {
			return nil, err
		}

		pl = consensusmsg.NewCandidate(blk)
		step = ctx.BlockStep
	case consensusmsg.BlockReductionID:
		sig, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, blockHash)
		if err != nil {
			return nil, err
		}

		if spoofSig {
			sig = make([]byte, 33)
		}

		pl, err = consensusmsg.NewBlockReduction(blockHash, sig,
			ctx.Keys.BLSPubKey.Marshal())
		if err != nil {
			return nil, err
		}

		step = ctx.BlockStep
	case consensusmsg.BlockAgreementID:
		pl, err = consensusmsg.NewBlockAgreement(blockHash, voteSet)
		if err != nil {
			return nil, err
		}

		step = ctx.BlockStep
	case consensusmsg.SigSetCandidateID:
		pl, err = consensusmsg.NewSigSetCandidate(blockHash, voteSet, 2)
		if err != nil {
			return nil, err
		}

		step = ctx.SigSetStep
	case consensusmsg.SigSetReductionID:
		sig, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, blockHash)
		if err != nil {
			return nil, err
		}

		if spoofSig {
			sig = make([]byte, 33)
		}

		pl, err = consensusmsg.NewSigSetReduction(blockHash, blockHash, sig,
			ctx.Keys.BLSPubKey.Marshal())
		if err != nil {
			return nil, err
		}

		step = ctx.SigSetStep
	}

	// Complete message and return it
	sigEd, err := ctx.CreateSignature(pl, step)
	if err != nil {
		return nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, c.LastHeader.Hash, step,
		sigEd, keys.EdPubKeyBytes(), pl)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
