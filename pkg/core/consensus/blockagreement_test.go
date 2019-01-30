package consensus

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Block agreement runs until a decision is made. Thus, we will only test for
// completion in this phase. As of right now, the phase is not timed.
func TestBlockAgreement(t *testing.T) {
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
	BlockAgreement(ctx, c)

	q <- true

	// Should have terminated successfully
	result := <-c
	assert.True(t, result)

	// Vote set should have remained the same
	assert.Equal(t, votes, ctx.BlockVotes)
}
