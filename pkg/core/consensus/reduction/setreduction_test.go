package reduction_test

import (
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

func TestSignatureSetReductionDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500
	ctx.VoteLimit = 100
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Run sortition
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Votes = votes

	// Create vote set
	voteSet, _, err := createVotesAndMsgs(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet

	// Run signature set reduction function
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := consensus.SignatureSetReduction(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait one second..
	time.Sleep(1 * time.Second)

	// SigSetHash should now be set
	// Make vote that will win by a large margin
	_, vote1, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	// Log current SigSetHash
	var currHash []byte
	currHash = append(currHash, ctx.SigSetHash...)

	// Send vote
	ctx.SigSetVoteChan <- vote1

	// Wait one second..
	time.Sleep(1 * time.Second)

	// Make new vote, send it, and wait for function to return
	_, vote2, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVoteChan <- vote2
	wg.Wait()

	// Function should have returned and we should have the same sigsethash
	assert.Equal(t, currHash, ctx.SigSetHash)
}

func TestSignatureSetReductionIndecisive(t *testing.T) {
	// Adjust timer to reduce waiting times
	consensus.StepTime = 1 * time.Second

	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500
	ctx.VoteLimit = 100
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Run sortition
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Votes = votes

	// Create vote set
	voteSet, _, err := createVotesAndMsgs(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet

	// The function will return as soon as it doesn't get a decisive outcome.
	if err := consensus.SignatureSetReduction(ctx); err != nil {
		t.Fatal(err)
	}

	// Function should have returned and we should have a nil sigsethash
	assert.Nil(t, ctx.SigSetHash)

	// Reset step timer
	consensus.StepTime = 20 * time.Second
}

func newVoteSigSet(c *consensus.Context, weight uint64, winningBlock, setHash []byte) (uint64, *payload.MsgConsensus, error) {
	if weight < sortition.MinimumStake {
		return 0, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := consensus.NewRandKeys()
	ctx, err := consensus.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return 0, nil, err
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

	// Run sortition
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		return 0, nil, err
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		return 0, nil, err
	}

	ctx.Votes = votes
	if ctx.Votes == 0 {
		return 0, nil, errors.New("sortition did not generate any votes")
	}

	// Sign sig set with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
	if err != nil {
		return 0, nil, err
	}

	// Set BLS key on context
	blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
	c.NodeBLS[hex.EncodeToString([]byte(*keys.EdPubKey))] = blsPubBytes

	// Create sigsetvote payload to gossip
	pl, err := consensusmsg.NewSigSetVote(winningBlock, ctx.SigSetHash, sigBLS, blsPubBytes, ctx.Score)
	if err != nil {
		return 0, nil, err
	}

	sigEd, err := consensus.CreateSignature(ctx, pl)
	if err != nil {
		return 0, nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return 0, nil, err
	}

	return ctx.Votes, msg, nil
}
