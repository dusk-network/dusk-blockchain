package collection_test

import (
	"sync"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"

	"github.com/stretchr/testify/assert"
)

func TestBlockCollection(t *testing.T) {
	// Lower candidate timer to prevent waiting times
	user.CandidateTime = 5 * time.Second

	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Make block candidate
	blk, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Add one dummy tx to block
	txs := ctx.GetAllTXs()
	blk.AddTx(txs[0])

	// Make candidate payload and message
	pl := consensusmsg.NewCandidate(blk)
	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		t.Fatal(err)
	}

	msgCandidate, err := payload.NewMsgConsensus(ctx.Version, ctx.Round,
		ctx.LastHeader.Hash, ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		t.Fatal(err)
	}

	// Make spoofed score
	// TODO: Make an actual score and have it pass verification
	proof, err := crypto.RandEntropy(100)
	if err != nil {
		t.Fatal(err)
	}

	pl2, err := consensusmsg.NewCandidateScore(300, proof, blk.Header.Hash, blk.Header.Seed)
	if err != nil {
		t.Fatal(err)
	}

	sigEd2, err := ctx.CreateSignature(pl2)
	if err != nil {
		t.Fatal(err)
	}

	msgScore, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd2, []byte(*ctx.Keys.EdPubKey), pl2)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := collection.Block(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send messages
	ctx.CandidateScoreChan <- msgScore
	ctx.CandidateChan <- msgCandidate

	// Block until block collection returns
	wg.Wait()

	// CandidateBlock and BlockHash should not be nil after receiving at least
	// one candidate and score message
	assert.NotNil(t, ctx.BlockHash)
	assert.NotNil(t, ctx.CandidateBlocks)

	// Reset candidate timer
	user.CandidateTime = 60 * time.Second
}

func TestBlockCollectionNoBlock(t *testing.T) {
	// Lower candidate timer to prevent waiting times
	user.CandidateTime = 1 * time.Second

	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Run block collection and let it time out
	if err := collection.Block(ctx); err != nil {
		t.Fatal(err)
	}

	// BlockHash and CandidateBlock should be nil
	assert.Nil(t, ctx.BlockHash)
	assert.Empty(t, ctx.CandidateBlocks)

	// Reset candidate timer
	user.CandidateTime = 60 * time.Second
}
