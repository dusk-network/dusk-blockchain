package consensus

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

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

	// Set basic fields on context
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	// Make vote set and vote messages
	votes, msgs, err := createVotesAndMsgs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = votes

	// Send msgs
	q := make(chan bool, 1)
	ticker := time.NewTicker(300 * time.Millisecond)
	go func() {
		for i := 0; i < len(msgs); i++ {
			select {
			case <-q:
				ticker.Stop()
				return
			case <-ticker.C:
				// First encode and decode to avoid pointer issues
				buf := new(bytes.Buffer)
				if err := msgs[i].Encode(buf); err != nil {
					t.Fatal(err)
				}

				msg := &payload.MsgConsensus{}
				if err := msg.Decode(buf); err != nil {
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

	// Set basic fields on context
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

	// Create a dummy vote set
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

		vote, err := consensusmsg.NewVote(hash, keys.BLSPubKey.Marshal(), sig, sig, 1)
		if err != nil {
			t.Fatal(err)
		}

		votes = append(votes, vote)
	}

	ctx.BlockVotes = votes

	// Send the set agreement message with the vote set we just created
	if err := sendSetAgreement(ctx, ctx.BlockVotes); err != nil {
		t.Fatal(err)
	}

	// Should have gotten the message into our setagreement channel
	msg := <-ctx.SetAgreementChan
	assert.NotNil(t, msg)
}

// Convenience function to make a vote set and messages from all voters.
// It should pass verifications done by the passed context object after the function returns,
// and all the messages should exceed the vote limit.
func createVotesAndMsgs(ctx *Context, amount int) ([]*consensusmsg.Vote,
	[]*payload.MsgConsensus, error) {
	var votes []*consensusmsg.Vote
	var ctxs []*Context

	// Make two votes per node
	for i := 0; i < amount; i++ {
		keys, err := NewRandKeys()
		if err != nil {
			return nil, nil, err
		}

		// Set these keys in our context values to pass processing
		pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
		pkEd := hex.EncodeToString([]byte(*keys.EdPubKey))
		ctx.NodeWeights[pkEd] = 500
		ctx.NodeBLS[pkBLS] = []byte(*keys.EdPubKey)

		// Make dummy context for score creation
		c, err := NewContext(0, 0, ctx.W, ctx.Round, ctx.Seed, ctx.Magic, keys)
		if err != nil {
			return nil, nil, err
		}

		c.LastHeader = ctx.LastHeader
		c.weight = 500
		c.BlockHash = ctx.BlockHash

		// Run sortition
		role := &role{
			part:  "committee",
			round: ctx.Round,
			step:  ctx.Step,
		}

		if err := sortition(c, role); err != nil {
			return nil, nil, err
		}

		// Create vote signatures
		sig1, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, nil, err
		}

		sig2, err := ctx.BLSSign(keys.BLSSecretKey, keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, nil, err
		}

		// Create two votes and add them to the array
		vote1, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), sig1,
			c.Score, ctx.Step)
		if err != nil {
			return nil, nil, err
		}

		vote2, err := consensusmsg.NewVote(ctx.BlockHash, keys.BLSPubKey.Marshal(), sig2,
			c.Score, ctx.Step-1)
		if err != nil {
			return nil, nil, err
		}

		votes = append(votes, vote1)
		votes = append(votes, vote2)
		ctxs = append(ctxs, c)
	}

	var msgs []*payload.MsgConsensus
	for _, c := range ctxs {
		// Do this to avoid pointer issues during processing
		newVotes := make([]*consensusmsg.Vote, 0)
		newVotes = append(newVotes, votes...)

		pl, err := consensusmsg.NewSetAgreement(c.BlockHash, newVotes)
		if err != nil {
			return nil, nil, err
		}

		sigEd, err := createSignature(c, pl)
		if err != nil {
			return nil, nil, err
		}

		msg, err := payload.NewMsgConsensus(c.Version, c.Round, c.LastHeader.Hash,
			c.Step, sigEd, []byte(*c.Keys.EdPubKey), pl)
		if err != nil {
			return nil, nil, err
		}

		msgs = append(msgs, msg)
	}

	return votes, msgs, nil
}
