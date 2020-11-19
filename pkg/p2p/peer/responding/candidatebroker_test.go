package responding_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test the functionality of the CandidateBroker.
func TestCandidateBroker(t *testing.T) {
	// Set up db
	_, db := lite.CreateDBConnection()
	defer func() {
		_ = db.Close()
	}()

	// Generate 5 candidates and store them in the db. Save the hashes for later checking.
	hashes, blocks := generateBlocks(5)
	assert.NoError(t, storeCandidates(db, blocks))

	c := responding.NewCandidateBroker(db)

	// First, ask for the wrong candidate.
	wrongHash, _ := crypto.RandEntropy(32)
	_, err := c.ProvideCandidate(message.New(topics.GetCandidate, message.GetCandidate{
		Hash: wrongHash},
	))
	assert.Error(t, err)

	// Now, ask for the correct one.
	buf, err := c.ProvideCandidate(message.New(topics.GetCandidate, message.GetCandidate{
		Hash: hashes[0]},
	))
	assert.NoError(t, err)

	// Remove topic from buffer
	_, _ = topics.Extract(&buf[0])

	// Ensure it is the same block
	cm := message.MakeCandidate(block.NewBlock())
	assert.NoError(t, message.UnmarshalCandidate(&buf[0], &cm))
	assert.True(t, cm.Block.Equals(blocks[0]))
}

func storeCandidates(db database.DB, blocks []*block.Block) error {
	return db.Update(func(t database.Transaction) error {
		for _, blk := range blocks {
			cm := message.NewCandidate()
			cm.Block = blk
			if err := t.StoreCandidateMessage(*cm); err != nil {
				return err
			}
		}
		return nil
	})
}
