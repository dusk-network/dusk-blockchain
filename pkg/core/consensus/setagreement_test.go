package consensus

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Signature set agreement runs until a decision is made. Thus, we will only test for
// completion in this phase. As of right now, the phase is not timed.
func TestSetAgreement(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.weight = 500
	ctx.VoteLimit = 20

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	// Make vote set with at least 2*VoteLimit amount of votes
	votes, err := createVoteSet(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockVotes = votes

	// This should conclude the signature set agreement phase fairly quick
	q := make(chan bool, 1)
	ticker := time.NewTicker(300 * time.Millisecond)
	go func() {
		for {
			select {
			case <-q:
				ticker.Stop()
				return
			case <-ticker.C:
				weight := rand.Intn(10000)
				weight += 100 // Avoid stakes being too low to participate
				newVotes, err := createVoteSet(ctx, 50)
				if err != nil {
					t.Fatal(err)
				}

				_, msg, err := newVoteSetAgreement(ctx, uint64(weight), candidateBlock, newVotes)
				if err != nil {
					t.Fatal(err)
				}

				ctx.SetAgreementChan <- msg
			}
		}
	}()

	c := make(chan bool, 1)
	SignatureSetAgreement(ctx, c)

	q <- true

	// Should have terminated successfully
	result := <-c
	assert.True(t, result)
}

// Test for set agreement convenience function
func TestSendSetAgreement(t *testing.T) {
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	prevHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.LastHeader.Hash = prevHash
	ctx.BlockHash = hash

	var votes []*consensusmsg.Vote
	for i := 0; i < 5; i++ {
		keys, err := NewRandKeys()
		if err != nil {
			t.Fatal(err)
		}

		sig, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, hash)
		if err != nil {
			t.Fatal(err)
		}

		vote, err := consensusmsg.NewVote(hash, keys.BLSPubKey.Marshal(), sig)
		if err != nil {
			t.Fatal(err)
		}

		votes = append(votes, vote)
	}

	ctx.BlockVotes = votes
	if err := sendSetAgreement(ctx); err != nil {
		t.Fatal(err)
	}

	// Should have gotten the message into our setagreement channel
	msg := <-ctx.SetAgreementChan

	assert.NotNil(t, msg)
}

func newVoteSetAgreement(c *Context, weight uint64, blockHash []byte,
	votes []*consensusmsg.Vote) (uint64, *payload.MsgConsensus, error) {
	if weight < MinimumStake {
		return 0, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewContext(0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return 0, nil, err
	}

	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)
	ctx.weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = blockHash
	ctx.Step = c.Step

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		return 0, nil, err
	}

	if ctx.votes > 0 {
		// Create setagreement payload to gossip
		pl, err := consensusmsg.NewSetAgreement(blockHash, votes)
		if err != nil {
			return 0, nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return 0, nil, err
		}

		return ctx.votes, msg, nil
	}

	return 0, nil, nil
}

func createVoteSet(ctx *Context, amount int) ([]*consensusmsg.Vote, error) {
	var votes []*consensusmsg.Vote
	for i := 0; i < amount; i++ {
		keys, err := NewRandKeys()
		if err != nil {
			return nil, err
		}

		// Set these keys in our context values to pass processing
		pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
		pkEd := hex.EncodeToString([]byte(*keys.EdPubKey))
		ctx.NodeWeights[pkEd] = 500
		ctx.NodeBLS[pkBLS] = []byte(*keys.EdPubKey)

		sig, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, err
		}

		cSig := sig.Compress()

		vote, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), cSig)
		if err != nil {
			return nil, err
		}

		votes = append(votes, vote)
	}

	return votes, nil
}
