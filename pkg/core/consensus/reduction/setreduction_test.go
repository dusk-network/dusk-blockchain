package reduction_test

import (
	"encoding/hex"
	"sync"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Test if votes during signature set reduction happen successfully.
func TestSignatureSetReductionVote(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	ctx.CandidateBlock = &block.Block{}

	// Create vote set
	voteSet, err := createVotes(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet
	setHash, err := ctx.HashVotes(voteSet)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetHash = setHash

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, ctx.Keys.EdPubKeyBytes())

	// Start block reduction with us as the only committee member
	if err := reduction.SignatureSet(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSignatureSetReductionDecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Create vote set
	voteSet, err := createVotes(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet
	setHash, err := ctx.HashVotes(voteSet)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetHash = setHash

	// Log current SigSetHash
	var currHash []byte
	currHash = append(currHash, ctx.SigSetHash...)

	// SigSetHash should now be set
	// Make vote that will win by a large margin
	vote1, vote2, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetReductionChan <- vote1
	ctx.SigSetReductionChan <- vote2

	// Run signature set reduction function
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.SignatureSet(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	wg.Wait()

	// Function should have returned and we should have the same sigsethash
	assert.Equal(t, currHash, ctx.SigSetHash)
}

func TestSignatureSetReductionIndecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Create vote set
	voteSet, err := createVotes(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet
	setHash, err := ctx.HashVotes(voteSet)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetHash = setHash

	// The function will return as soon as it doesn't get a decisive outcome.
	if err := reduction.SignatureSet(ctx); err != nil {
		t.Fatal(err)
	}

	// Function should have returned and we should have a nil sigsethash
	assert.Equal(t, make([]byte, 32), ctx.SigSetHash)
}

func newVoteSigSet(c *user.Context, weight uint64, winningBlock,
	setHash []byte) (*payload.MsgConsensus, *payload.MsgConsensus, error) {
	// Create context
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, nil, err
	}

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString(keys.EdPubKeyBytes())] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = keys.EdPubKeyBytes()

	// Populate new context fields
	ctx.Weight = weight
	c.W += weight
	ctx.LastHeader = c.LastHeader
	ctx.SigSetHash = setHash
	ctx.BlockHash = winningBlock
	ctx.SigSetStep = c.SigSetStep

	// Add to our committee
	c.Committee = append(c.Committee, keys.EdPubKeyBytes())

	// Sign sig set with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
	if err != nil {
		return nil, nil, err
	}

	// Set BLS key on context
	blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
	c.NodeBLS[hex.EncodeToString(keys.EdPubKeyBytes())] = blsPubBytes

	// Create sigsetvote payload to gossip
	pl, err := consensusmsg.NewSigSetReduction(winningBlock, ctx.SigSetHash, sigBLS, blsPubBytes)
	if err != nil {
		return nil, nil, err
	}

	sigEd, err := ctx.CreateSignature(pl, ctx.SigSetStep)
	if err != nil {
		return nil, nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.SigSetStep, sigEd, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return nil, nil, err
	}

	sigEd2, err := ctx.CreateSignature(pl, ctx.SigSetStep+1)
	if err != nil {
		return nil, nil, err
	}

	msg2, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.SigSetStep+1, sigEd2, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return nil, nil, err
	}

	return msg, msg2, nil
}

func createVotes(ctx *user.Context, amount int) ([]*consensusmsg.Vote, error) {
	var voteSet []*consensusmsg.Vote
	for i := 0; i < amount; i++ {
		keys, err := user.NewRandKeys()
		if err != nil {
			return nil, err
		}

		// Set these keys in our context values to pass processing
		pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
		pkEd := hex.EncodeToString(keys.EdPubKeyBytes())
		ctx.NodeWeights[pkEd] = 500
		ctx.NodeBLS[pkBLS] = keys.EdPubKeyBytes()

		// Make dummy context for score creation
		c, err := user.NewContext(0, 0, ctx.W, ctx.Round, ctx.Seed, ctx.Magic, keys)
		if err != nil {
			return nil, err
		}

		c.LastHeader = ctx.LastHeader
		c.Weight = 500
		c.BlockHash = ctx.BlockHash

		// Create vote signatures
		sig1, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, err
		}

		sig2, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, err
		}

		// Create two votes and add them to the array
		vote1, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), sig1,
			ctx.SigSetStep)
		if err != nil {
			return nil, err
		}

		vote2, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), sig2,
			ctx.SigSetStep-1)
		if err != nil {
			return nil, err
		}

		voteSet = append(voteSet, vote1)
		voteSet = append(voteSet, vote2)
	}

	return voteSet, nil
}
