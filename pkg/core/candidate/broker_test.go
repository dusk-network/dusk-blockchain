package candidate_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	assert "github.com/stretchr/testify/require"
)

func TestBroker(t *testing.T) {
	_, db := lite.CreateDBConnection()
	b := candidate.NewBroker(db)

	cm := mockCandidate()
	b.StoreCandidateMessage(message.New(topics.Candidate, cm))

	// DB should have the candidate now
	assert.NoError(t, db.View(func(tr database.Transaction) error {
		c, err := tr.FetchCandidateMessage(cm.Block.Header.Hash)
		if err != nil {
			return err
		}

		assert.True(t, c.Block.Equals(cm.Block))
		return nil
	}))
}

// Mocks a candidate message. It is not in the message package since it uses
// the genesis block as mockup block
//nolint:unused
func mockCandidate() message.Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	return message.MakeCandidate(genesis, cert)
}
