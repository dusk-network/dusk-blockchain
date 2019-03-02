package msg_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestVerifyBlockAgreement(t *testing.T) {
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

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	if err := ctx.SetCommittee(ctx.BlockStep); err != nil {
		t.Fatal(err)
	}

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestBlockAgreementNotInCommittee(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Add new members to committee
	for i := 0; i < 20; i++ {
		newMember, err := crypto.RandEntropy(32)
		if err != nil {
			t.Fatal(err)
		}

		ctx.Committee = append(ctx.Committee, newMember)
		pkEd := hex.EncodeToString(newMember)
		ctx.NodeWeights[pkEd] = 500
		ctx.W += 500
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("committee check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementSmallVoteSet(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote set size check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterStakeCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Cut out node weights except for message sender
	pkEd := hex.EncodeToString(m.PubKey)
	for node := range ctx.NodeWeights {
		if node == pkEd {
			continue
		}

		ctx.NodeWeights[node] = 0
	}

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote stake check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterBLSCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Cut out node weights except for message sender
	for node, pkBytes := range ctx.NodeBLS {
		if bytes.Equal(m.PubKey, pkBytes) {
			continue
		}

		ctx.NodeBLS[node] = make([]byte, 32)
	}

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote BLS check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterCommitteeCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Cut out some committee members except for message sender
	for i, pkBytes := range ctx.Committee {
		if bytes.Equal(pkBytes, m.PubKey) {
			continue
		}

		if i%2 == 0 {
			ctx.Committee = append(ctx.Committee[i:], ctx.Committee[:i+1]...)
		}
	}

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote BLS check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterSigCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, true, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterStepCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestBlockAgreementVoterHashCheck(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 5000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy block and message
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.BlockStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x03, false, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// Set up committee
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.BlockStep,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		t.Fatal(err)
	}

	ctx.CurrentCommittee = committee

	// Verify the message
	err2 := msg.Process(ctx, m)
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

// Convenience function to create a vote set that can be verified reliably,
// or can be altered to have specific checks fail.
func createVoteSetAndMsg(ctx *user.Context, blockHash []byte, amount int,
	id uint8, spoofSig, spoofStep, spoofHash bool) (*payload.MsgConsensus, error) {
	var ctxs []*user.Context
	for i := 0; i < amount; i++ {
		keys, err := user.NewRandKeys()
		if err != nil {
			return nil, err
		}

		// Set these keys in our context values to pass processing
		pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
		pkEd := hex.EncodeToString([]byte(*keys.EdPubKey))
		ctx.NodeWeights[pkEd] = 500
		ctx.NodeBLS[pkBLS] = []byte(*keys.EdPubKey)
		ctx.Committee = append(ctx.Committee, []byte(*keys.EdPubKey))

		// Make dummy context for score creation
		c, err := user.NewContext(0, 0, ctx.W, ctx.Round, ctx.Seed, ctx.Magic, keys)
		if err != nil {
			return nil, err
		}

		c.LastHeader = ctx.LastHeader
		c.Weight = 500
		ctx.W += 500
		c.BlockHash = blockHash

		ctxs = append(ctxs, c)
	}

	committee1, err := sortition.CreateCommittee(ctx.Round, ctx.W, 1, ctx.Committee,
		ctx.NodeWeights)
	if err != nil {
		return nil, err
	}

	committee2, err := sortition.CreateCommittee(ctx.Round, ctx.W, 2, ctx.Committee,
		ctx.NodeWeights)
	if err != nil {
		return nil, err
	}

	voteSet, err := createVotes(committee1, committee2, ctxs, blockHash, spoofSig, spoofStep, spoofHash)
	if err != nil {
		return nil, err
	}

	// Make message from one of the context objects we just created
	var pl consensusmsg.Msg
	pk := committee2[1]
	var sendCtx *user.Context
	for _, c := range ctxs {
		if bytes.Equal([]byte(*c.Keys.EdPubKey), pk) {
			sendCtx = c
			break
		}
	}
	switch consensusmsg.ID(id) {
	case consensusmsg.BlockAgreementID:
		pl, err = consensusmsg.NewBlockAgreement(blockHash, voteSet)
		if err != nil {
			return nil, err
		}
	case consensusmsg.SigSetCandidateID:
		pl, err = consensusmsg.NewSigSetCandidate(blockHash, voteSet, 2)
		if err != nil {
			return nil, err
		}
	case consensusmsg.SigSetAgreementID:
		hash, err := sendCtx.HashVotes(voteSet)
		if err != nil {
			return nil, err
		}

		// Adjust votes to vote for the hash instead
		voteSet, err := createVotes(committee1, committee2, ctxs, hash, spoofSig, spoofStep, spoofHash)
		if err != nil {
			return nil, err
		}

		pl, err = consensusmsg.NewSigSetAgreement(blockHash, hash, voteSet, 2)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("wrong id for vote set and message creation")
	}

	sigEd, err := sendCtx.CreateSignature(pl, 2)
	if err != nil {
		return nil, err
	}

	m, err := payload.NewMsgConsensus(sendCtx.Version, sendCtx.Round, sendCtx.LastHeader.Hash, 2,
		sigEd, []byte(*sendCtx.Keys.EdPubKey), pl)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func createVotes(committee1, committee2 [][]byte, ctxs []*user.Context,
	hash []byte, spoofSig, spoofStep, spoofHash bool) ([]*consensusmsg.Vote, error) {
	var voteSet []*consensusmsg.Vote

	for _, pk := range committee1 {
		for _, user := range ctxs {
			if !bytes.Equal(pk, []byte(*user.Keys.EdPubKey)) {
				continue
			}

			sig, err := user.BLSSign(user.Keys.BLSSecretKey, user.Keys.BLSPubKey, hash)
			if err != nil {
				return nil, err
			}

			if spoofSig {
				sig = make([]byte, 33)
			}

			vote, err := consensusmsg.NewVote(hash, user.Keys.BLSPubKey.Marshal(), sig, 1)
			if err != nil {
				return nil, err
			}

			if spoofStep {
				vote.Step += 2
			}

			if spoofHash {
				vote.Hash = make([]byte, 32)
			}

			voteSet = append(voteSet, vote)
		}
	}

	for _, pk := range committee2 {
		for _, user := range ctxs {
			if !bytes.Equal(pk, []byte(*user.Keys.EdPubKey)) {
				continue
			}

			sig, err := user.BLSSign(user.Keys.BLSSecretKey, user.Keys.BLSPubKey, hash)
			if err != nil {
				return nil, err
			}

			if spoofSig {
				sig = make([]byte, 33)
			}

			vote, err := consensusmsg.NewVote(hash, user.Keys.BLSPubKey.Marshal(), sig, 2)
			if err != nil {
				return nil, err
			}

			if spoofStep {
				vote.Step += 2
			}

			if spoofHash {
				vote.Hash = make([]byte, 32)
			}

			voteSet = append(voteSet, vote)
		}
	}

	return voteSet, nil
}
