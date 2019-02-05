package consensus_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Block agreement runs until a decision is made. Thus, we will only test for
// completion in this phase. As of right now, the phase is not timed.
func TestBlockAgreement(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic parameters
	ctx.Weight = 500
	ctx.VoteLimit = 20

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	// Make vote set and vote messages
	votes, msgs, err := createVotesAndMsgs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockVotes = votes

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
	consensus.BlockAgreement(ctx, c)

	q <- true

	// Should have terminated successfully
	result := <-c
	assert.True(t, result)

	// Vote set should have remained the same
	assert.Equal(t, votes, ctx.BlockVotes)
}
