package msg_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestVerifySigSetAgreement(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash
	ctx.BlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, ctx.BlockHash, 10, 0x06, false, false, false)
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

func TestSigSetAgreementNotInCommittee(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, false)
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
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.SigSetStep,
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

func TestSigSetAgreementWrongBlock(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set with a different hash
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, hash, 10, 0x06, false, false, false)
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
	if err2 == nil {
		t.Fatal("block hash check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetAgreementSmallVoteSet(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Increase committee
	for i := 0; i < 20; i++ {
		ctx.Committee = append(ctx.Committee, make([]byte, 32))
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
	if err2 == nil {
		t.Fatal("vote set size check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetAgreementVoterStakeCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, false)
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

func TestSigSetAgreementVoterBLSCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, false)
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

func TestSigSetAgreementVoterCommitteeCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, false)
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

func TestSigSetAgreementVoterSigCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, true, false, false)
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
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetAgreementVoterStepCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, true, false)
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
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}

func TestSigSetAgreementVoterHashCheck(t *testing.T) {
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

	ctx.WinningBlockHash = emptyBlock.Header.Hash

	// Create vote set
	ctx.SigSetStep++
	m, err := createVoteSetAndMsg(ctx, emptyBlock.Header.Hash, 10, 0x06, false, false, true)
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
	if err2 == nil {
		t.Fatal("vote sig check did not work")
	}

	if err2.Priority == prerror.High {
		t.Fatal(err2)
	}
}
