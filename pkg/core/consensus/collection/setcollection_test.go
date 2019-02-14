package collection_test

import (
	"bytes"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// This test will test signature set collection.
func TestSignatureSetCollection(t *testing.T) {
	// Put step timer down to avoid long waiting times
	user.StepTime = 3 * time.Second

	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = hash
	voteSet, err := createVotes(ctx, 15)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet
	ctx.Weight = 200

	// Shuffle vote set
	var otherVotes []*consensusmsg.Vote
	otherVotes = append(otherVotes, voteSet...)
	otherVotes[0] = otherVotes[1]

	// Create votes
	vote1, err := newSigSetCandidate(ctx, 500, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	vote2, err := newSigSetCandidate(ctx, 800, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	vote3, err := newSigSetCandidate(ctx, 1500, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	allVotes := []*payload.MsgConsensus{vote1, vote2, vote3}

	// Run signature set collections
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := collection.SignatureSet(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send votes, and then block until function returns
	// The order should not matter, as the highest stake will win
	for _, vote := range allVotes {
		// Do this to avoid pointer issues during verification
		buf := new(bytes.Buffer)
		if err := vote.Encode(buf); err != nil {
			t.Fatal(err)
		}

		msg := &payload.MsgConsensus{}
		if err := msg.Decode(buf); err != nil {
			t.Fatal(err)
		}

		ctx.SigSetCandidateChan <- msg
	}
	wg.Wait()

	// We should now have otherVotes
	assert.Equal(t, otherVotes, ctx.SigSetVotes)

	// Reset step timer
	user.StepTime = 20 * time.Second
}

func newSigSetCandidate(c *user.Context, weight uint64,
	voteSet []*consensusmsg.Vote) (*payload.MsgConsensus, error) {
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
	ctx.BlockHash = c.BlockHash
	ctx.Step = c.Step
	ctx.SigSetVotes = voteSet

	// Add to our committee
	c.CurrentCommittee = append(c.CurrentCommittee, []byte(*keys.EdPubKey))

	// Create payload, signature and message
	pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes)
	if err != nil {
		return nil, err
	}

	sigEd, err := ctx.CreateSignature(pl)
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
