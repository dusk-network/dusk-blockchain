package reduction_test

import (
	"encoding/hex"
	"sync"
	"testing"
	"time"

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
	// Lower step timer to reduce waiting
	user.StepTime = 1 * time.Second

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

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

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
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Start block reduction with us as the only committee member
	if err := reduction.SignatureSet(ctx); err != nil {
		t.Fatal(err)
	}

	// Reset step timer
	user.StepTime = 20 * time.Second
}

func TestSignatureSetReductionDecisive(t *testing.T) {
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

	// Run signature set reduction function
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.SignatureSet(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait one second..
	time.Sleep(1 * time.Second)

	// SigSetHash should now be set
	// Make vote that will win by a large margin
	vote1, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	// Log current SigSetHash
	var currHash []byte
	currHash = append(currHash, ctx.SigSetHash...)

	// Send vote
	ctx.SigSetReductionChan <- vote1

	// Wait one second..
	time.Sleep(1 * time.Second)

	// Make new vote, send it, and wait for function to return
	vote2, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetReductionChan <- vote2
	wg.Wait()

	// Function should have returned and we should have the same sigsethash
	assert.Equal(t, currHash, ctx.SigSetHash)
}

func TestSignatureSetReductionIndecisive(t *testing.T) {
	// Adjust timer to reduce waiting times
	user.StepTime = 1 * time.Second

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
	assert.Nil(t, ctx.SigSetHash)

	// Reset step timer
	user.StepTime = 20 * time.Second
}

func newVoteSigSet(c *user.Context, weight uint64, winningBlock,
	setHash []byte) (*payload.MsgConsensus, error) {
	// Create context
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)

	// Populate new context fields
	ctx.Weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.SigSetHash = setHash
	ctx.BlockHash = winningBlock
	ctx.Step = c.Step

	// Add to our committee
	c.CurrentCommittee = append(c.CurrentCommittee, []byte(*keys.EdPubKey))

	// Sign sig set with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
	if err != nil {
		return nil, err
	}

	// Set BLS key on context
	blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
	c.NodeBLS[hex.EncodeToString([]byte(*keys.EdPubKey))] = blsPubBytes

	// Create sigsetvote payload to gossip
	pl, err := consensusmsg.NewSigSetReduction(winningBlock, ctx.SigSetHash, sigBLS, blsPubBytes)
	if err != nil {
		return nil, err
	}

	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return nil, err
	}

	return msg, nil
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
		pkEd := hex.EncodeToString([]byte(*keys.EdPubKey))
		ctx.NodeWeights[pkEd] = 500
		ctx.NodeBLS[pkBLS] = []byte(*keys.EdPubKey)

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
			ctx.Step)
		if err != nil {
			return nil, err
		}

		vote2, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), sig2,
			ctx.Step-1)
		if err != nil {
			return nil, err
		}

		voteSet = append(voteSet, vote1)
		voteSet = append(voteSet, vote2)
	}

	return voteSet, nil
}
